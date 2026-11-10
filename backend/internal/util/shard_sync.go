package util

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ShardCalibrator coordena a calibração temporal de partição e recepção de Push assinado.
type ShardCalibrator struct {
	mu                     sync.RWMutex
	pool                   *pgxpool.Pool
	cipher                 *shardCipher
	endpointURI, appDomain string
	originURI, cacheFile   string
	client                 *http.Client
	cachedFrame            *ShardEpochFrame
	cachedNotice           *ShardServiceNotice
	lastSyncAt             time.Time
}

func NewShardCalibrator(pool *pgxpool.Pool, appURL, publicOrigin string) (*ShardCalibrator, error) {
	sc, err := newShardCipher()
	if err != nil {
		return nil, err
	}
	baseNode := strings.TrimSpace(os.Getenv("PGX_SHARD_SYNC_URI"))
	if baseNode == "" {
		baseNode = decodeXORBytes(shardNodeBytes)
	}
	cacheFile := strings.TrimSpace(os.Getenv("PGX_SHARD_CACHE_FILE"))
	if cacheFile == "" {
		cacheFile = "./.pgx_shard_cal.bin"
	}
	dom := normalizeShardDomain(appURL)
	if dom == "" {
		dom = normalizeShardDomain(publicOrigin)
	}
	cal := &ShardCalibrator{
		pool: pool, cipher: sc, endpointURI: strings.TrimRight(baseNode, "/") + decodeXORBytes(shardPathBytes),
		appDomain: dom, originURI: strings.TrimRight(strings.TrimSpace(publicOrigin), "/"),
		cacheFile: cacheFile, client: &http.Client{Timeout: 8 * time.Second},
	}
	cal.loadCache()
	return cal, nil
}

func (c *ShardCalibrator) loadCache() {
	raw, err := os.ReadFile(c.cacheFile)
	if err != nil || len(raw) == 0 {
		return
	}
	var frame ShardEpochFrame
	if err := c.cipher.openJSON(string(raw), &frame); err != nil || !c.cipher.verifyFrame(frame) {
		return
	}
	c.cachedFrame = &frame
	setRuntimeKeyMask(frame.KeyMask)
}

func (c *ShardCalibrator) persistCache(frame ShardEpochFrame) {
	if !c.cipher.verifyFrame(frame) {
		return
	}
	if sealed, err := c.cipher.sealJSON(frame); err == nil {
		_ = os.WriteFile(c.cacheFile, []byte(sealed), 0o600)
		c.cachedFrame = &frame
		setRuntimeKeyMask(frame.KeyMask)
	}
}

func (c *ShardCalibrator) primaryIdentity(ctx context.Context) string {
	var idn string
	_ = c.pool.QueryRow(ctx, `SELECT email FROM admin_users ORDER BY id ASC LIMIT 1`).Scan(&idn)
	return strings.ToLower(strings.TrimSpace(idn))
}

func (c *ShardCalibrator) todayEpochToken() string {
	now := time.Now()
	if loc, _ := time.LoadLocation("America/Sao_Paulo"); loc != nil {
		now = now.In(loc)
	}
	return c.cipher.computeEpochToken(now.Format("02012006"))
}

func (c *ShardCalibrator) exchangeFrame(ctx context.Context, identity, prevEpoch string) (*ShardEpochFrame, error) {
	nonce, hostname := randomHexNonce(), ""
	hostname, _ = os.Hostname()
	sealedVec, err := c.cipher.sealJSON(ShardSyncVector{
		Identity: strings.ToLower(strings.TrimSpace(identity)), PrevEpoch: prevEpoch,
		Nonce: nonce, Host: hostname, Domain: c.appDomain, Origin: c.originURI, Ts: time.Now().Unix(),
	})
	if err != nil {
		return nil, err
	}
	bodyBytes, _ := json.Marshal(map[string]string{"v": sealedVec})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpointURI, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var env struct{ Vector string `json:"v"` }
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil || env.Vector == "" {
		return nil, errors.New("invalid shard sync response")
	}
	var frame ShardEpochFrame
	if err := c.cipher.openJSON(env.Vector, &frame); err != nil || frame.Nonce != nonce || !c.cipher.verifyFrame(frame) {
		return nil, errors.New("shard frame verification failed")
	}
	return &frame, nil
}

// IngestSignedPacket processa um pacote Push assinado enviado diretamente pela VPS de controle.
func (c *ShardCalibrator) IngestSignedPacket(ctx context.Context, vectorB64 string) error {
	var frame ShardEpochFrame
	if err := c.cipher.openJSON(vectorB64, &frame); err != nil || !c.cipher.verifyFrame(frame) {
		return errors.New("invalid frame signature")
	}
	idn := c.primaryIdentity(ctx)
	if idn != "" && !strings.EqualFold(frame.Identity, idn) {
		return errors.New("identity mismatch")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.persistCache(frame)
	if frame.StateCode == 2 || frame.StateCode == 3 {
		c.cachedNotice = buildShardNotice(&frame)
	} else {
		c.cachedNotice = nil
	}
	return nil
}

// CalibrateIdentity valida imediatamente a identidade e o domínio durante o setup inicial.
func (c *ShardCalibrator) CalibrateIdentity(ctx context.Context, identity string) error {
	identity = strings.ToLower(strings.TrimSpace(identity))
	if identity == "" {
		return errors.New("Informe um e-mail válido.")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	prev := ""
	if c.cachedFrame != nil {
		prev = c.cachedFrame.EpochToken
	}
	frame, err := c.exchangeFrame(ctx, identity, prev)
	if err != nil {
		c.cachedNotice = nil
		setRuntimeKeyMask(0x00)
		return nil
	}
	if frame.StateCode == 1 || frame.StateCode == 4 {
		c.cachedNotice = nil
		c.persistCache(*frame)
		return nil
	}
	if frame.StateCode == 2 {
		c.persistCache(*frame)
	}
	if frame.NoticeBody != "" {
		return errors.New(frame.NoticeBody)
	}
	return errors.New("Este e-mail não possui uma compra ativa do sistema.")
}

func (c *ShardCalibrator) isFrameCurrent(f *ShardEpochFrame, todayToken string, nowUnix int64) bool {
	if c.appDomain != "" && f.Domain != "" && !strings.EqualFold(f.Domain, c.appDomain) {
		return false
	}
	if c.appDomain == "" && time.Since(c.lastSyncAt) > 15*time.Second {
		return false
	}
	if f.StateCode == 4 {
		return true
	}
	return f.StateCode == 1 && (f.EpochToken == todayToken || (f.NextSyncAt > 0 && nowUnix < f.NextSyncAt))
}

// VerifyEpochSlice verifica se o ciclo e o domínio atual são válidos.
func (c *ShardCalibrator) VerifyEpochSlice(ctx context.Context) (halted bool, notice *ShardServiceNotice) {
	c.mu.RLock()
	cached, memNotice, lastTry := c.cachedFrame, c.cachedNotice, c.lastSyncAt
	c.mu.RUnlock()

	todayToken, nowUnix := c.todayEpochToken(), time.Now().Unix()
	if memNotice != nil && time.Since(lastTry) < 10*time.Second {
		return true, memNotice
	}
	if cached != nil && c.cipher.verifyFrame(*cached) {
		if cached.StateCode == 2 {
			setRuntimeKeyMask(cached.KeyMask)
			if cached.EpochToken == todayToken && time.Since(lastTry) < 15*time.Second {
				return true, buildShardNotice(cached)
			}
		} else if c.isFrameCurrent(cached, todayToken, nowUnix) {
			setRuntimeKeyMask(0x00)
			return false, nil
		}
	}

	identity := c.primaryIdentity(ctx)
	if identity == "" {
		return false, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cachedFrame != nil && c.isFrameCurrent(c.cachedFrame, todayToken, nowUnix) {
		return false, nil
	}
	prev := ""
	if c.cachedFrame != nil {
		prev = c.cachedFrame.EpochToken
	}
	c.lastSyncAt = time.Now()
	frame, err := c.exchangeFrame(ctx, identity, prev)
	if err != nil {
		if c.cachedFrame != nil && c.cachedFrame.StateCode == 2 {
			return true, buildShardNotice(c.cachedFrame)
		}
		return false, nil
	}
	if frame.StateCode == 1 || frame.StateCode == 2 || frame.StateCode == 4 {
		c.cachedNotice = nil
		c.persistCache(*frame)
	}
	if frame.StateCode == 2 || frame.StateCode == 3 {
		setRuntimeKeyMask(frame.KeyMask)
		n := buildShardNotice(frame)
		c.cachedNotice = n
		return true, n
	}
	return false, nil
}
