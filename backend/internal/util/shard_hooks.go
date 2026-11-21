package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

type ShardHaltHook func(halted bool, mask byte)

var (
	shardHaltedFlag   atomic.Bool
	activeInstanceUID atomic.Value
	shardHooksMu      sync.RWMutex
	shardHaltHooks    []ShardHaltHook
)

// RegisterShardHaltHook registra um callback acionado sempre que o estado de bloqueio muda.
func RegisterShardHaltHook(fn ShardHaltHook) {
	if fn == nil {
		return
	}
	shardHooksMu.Lock()
	shardHaltHooks = append(shardHaltHooks, fn)
	shardHooksMu.Unlock()
}

func notifyShardState(halted bool, mask byte) {
	prevHalted := shardHaltedFlag.Swap(halted)
	prevMask := ActiveKeyMask()
	setRuntimeKeyMask(mask)
	if prevHalted == halted && prevMask == mask {
		return
	}
	shardHooksMu.RLock()
	hooks := append([]ShardHaltHook(nil), shardHaltHooks...)
	shardHooksMu.RUnlock()
	for _, fn := range hooks {
		fn(halted, mask)
	}
}

// IsShardHalted informa se a instância encontra-se bloqueada em memória (API ou Worker).
func IsShardHalted() bool {
	return shardHaltedFlag.Load()
}

// ApplyRemoteShardState aplica estado recebido via Redis Pub/Sub no processo Worker.
func ApplyRemoteShardState(halted bool, mask byte) {
	notifyShardState(halted, mask)
}

// ActiveInstanceUID retorna o identificador único persistente desta instalação.
func ActiveInstanceUID() string {
	if v, ok := activeInstanceUID.Load().(string); ok {
		return v
	}
	return ""
}

func loadOrCreateInstanceUID(cacheFile string) string {
	if existing := ActiveInstanceUID(); existing != "" {
		return existing
	}
	dir := filepath.Dir(cacheFile)
	uidPath := filepath.Join(dir, ".pgx_shard_uid.bin")
	if raw, err := os.ReadFile(uidPath); err == nil {
		if clean := sanitizeUIDToken(string(raw)); len(clean) >= 16 {
			activeInstanceUID.Store(clean)
			return clean
		}
	}
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	generated := "uid_" + hex.EncodeToString(b)
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(uidPath, []byte(generated), 0o600)
	activeInstanceUID.Store(generated)
	return generated
}

func sanitizeUIDToken(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) > 64 {
		s = s[:64]
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return ""
		}
	}
	return s
}

// InitWorkerShardState carrega o estado assinado inicial e o UID no boot do Worker.
func InitWorkerShardState(cacheFile string) {
	if strings.TrimSpace(cacheFile) == "" {
		cacheFile = "./.pgx_shard_cal.bin"
	}
	loadOrCreateInstanceUID(cacheFile)
	sc, err := newShardCipher()
	if err != nil {
		return
	}
	raw, err := os.ReadFile(cacheFile)
	if err != nil || len(raw) == 0 {
		return
	}
	var frame ShardEpochFrame
	if sc.openJSON(string(raw), &frame) != nil || !sc.verifyFrame(frame) {
		return
	}
	halted := frame.StateCode == 2 || frame.StateCode == 3
	notifyShardState(halted, frame.KeyMask)
}

// ResolveGatewayNodeURL retorna a URL da Central, bloqueando overrides por env em produção.
func ResolveGatewayNodeURL() string {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		if u := strings.TrimSpace(os.Getenv("PGX_SHARD_SYNC_URI")); u != "" {
			return strings.TrimRight(u, "/")
		}
	}
	return strings.TrimRight(decodeXORBytes(shardNodeBytes), "/")
}

var multiLabelTLDs = []string{
	".com.br", ".net.br", ".org.br", ".gov.br", ".edu.br", ".emp.br", ".ind.br",
	".co.uk", ".com.ar", ".com.pt", ".com.mx", ".com.co", ".sslip.io", ".nip.io",
}

func extractRegistrableBaseDomain(raw string) string {
	h := strings.ToLower(strings.TrimSpace(raw))
	if strings.Contains(h, "://") {
		if u, err := url.Parse(h); err == nil && u.Hostname() != "" {
			h = strings.ToLower(u.Hostname())
		}
	}
	if hostOnly, _, err := net.SplitHostPort(h); err == nil && hostOnly != "" {
		h = hostOnly
	}
	if idx := strings.IndexByte(h, '/'); idx != -1 {
		h = h[:idx]
	}
	h = strings.TrimPrefix(strings.TrimSuffix(h, "."), "www.")
	if h == "" || h == "localhost" || !strings.Contains(h, ".") {
		return ""
	}
	if ip := net.ParseIP(h); ip != nil {
		return ""
	}
	parts := strings.Split(h, ".")
	if len(parts) <= 2 {
		return h
	}
	for _, sfx := range multiLabelTLDs {
		if strings.HasSuffix(h, sfx) {
			if len(parts) >= 3 {
				return strings.Join(parts[len(parts)-3:], ".")
			}
			return h
		}
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// MatchesBoundDomain valida se o Host/Origin público pertence ao mesmo domínio base configurado.
func MatchesBoundDomain(boundDomain, candidateHostOrOrigin string) bool {
	boundBase := extractRegistrableBaseDomain(boundDomain)
	if boundBase == "" {
		return true
	}
	candBase := extractRegistrableBaseDomain(candidateHostOrOrigin)
	if candBase == "" {
		return true
	}
	return strings.EqualFold(boundBase, candBase)
}

// VerifyRequestHost garante que o Host/Origin HTTP das rotas administrativas corresponde ao domínio vinculado.
func (c *ShardCalibrator) VerifyRequestHost(reqHost, origin string) *ShardServiceNotice {
	c.mu.RLock()
	f := c.cachedFrame
	c.mu.RUnlock()
	if f == nil || f.Domain == "" {
		return nil
	}
	if !MatchesBoundDomain(f.Domain, reqHost) || (origin != "" && !MatchesBoundDomain(f.Domain, origin)) {
		return &ShardServiceNotice{
			Active: true, Badge: "Domínio Exclusivo", Title: "Domínio não autorizado para esta instância",
			Body: fmt.Sprintf("Esta instalação pertence exclusivamente ao domínio %s. O acesso administrativo por este endereço não está autorizado.", f.Domain),
			Items: []string{fmt.Sprintf("Acesse o painel apenas pelo domínio principal configurado (%s).", f.Domain)},
		}
	}
	return nil
}

func buildOfflineShardNotice() *ShardServiceNotice {
	return &ShardServiceNotice{
		Active: true, Badge: "Sincronização Pendente", Title: "Falha de comunicação com o gateway central",
		Body: "Esta instância não conseguiu sincronizar seu estado dentro da janela permitida. Verifique a conectividade de rede e o DNS da VPS.",
	}
}
