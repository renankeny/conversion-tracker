package util

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

const (
	shardPubVector = "4720cec54d85cd0c9d601cd16a9264bd0e94dfd387e9ad8c35a2330efc482c77"
	shardAESVector = "31287d7e5bbcd888fe8ac427e4513abeaba182b767598613b437faf7f9e44352"
)

var (
	shardXORMask   = []byte{0x5A, 0x3C, 0x9F, 0x12, 0xE7, 0x44, 0xA8, 0x61}
	shardNodeBytes = []byte{0x32, 0x48, 0xeb, 0x62, 0x94, 0x7e, 0x87, 0x4e, 0x39, 0x54, 0xfa, 0x71, 0x8c, 0x6a, 0xdc, 0x13, 0x3b, 0x5f, 0xf4, 0x77, 0x86, 0x20, 0xc7, 0x4f, 0x39, 0x53, 0xf2, 0x3c, 0x85, 0x36}
	shardPathBytes = []byte{0x75, 0x5d, 0xef, 0x7b, 0xc8, 0x32, 0x99, 0x4e, 0x2e, 0x59, 0xf3, 0x77, 0x8a, 0x21, 0xdc, 0x13, 0x23, 0x13, 0xec, 0x7a, 0x86, 0x36, 0xcc, 0x4c, 0x29, 0x45, 0xf1, 0x71}
	runtimeKeyMask uint32
)

func decodeXORBytes(src []byte) string {
	out := make([]byte, len(src))
	for i := range src {
		out[i] = src[i] ^ shardXORMask[i%len(shardXORMask)]
	}
	return string(out)
}

func normalizeShardDomain(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
			return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
		}
	}
	if idx := strings.IndexByte(raw, '/'); idx != -1 {
		raw = raw[:idx]
	}
	if idx := strings.IndexByte(raw, ':'); idx != -1 {
		raw = raw[:idx]
	}
	dom := strings.TrimPrefix(raw, "www.")
	if dom == "localhost" || dom == "127.0.0.1" || dom == "::1" {
		return ""
	}
	return dom
}

// ActiveKeyMask retorna a máscara de derivação de entropia ativa em memória.
func ActiveKeyMask() byte {
	return byte(atomic.LoadUint32(&runtimeKeyMask))
}

func setRuntimeKeyMask(mask byte) {
	atomic.StoreUint32(&runtimeKeyMask, uint32(mask))
}

// ApplyShardKeyMask aplica a máscara de calibração de partição sobre um segredo em memória.
func ApplyShardKeyMask(secret []byte) []byte {
	mask := ActiveKeyMask()
	if mask == 0 {
		return secret
	}
	derived := make([]byte, len(secret))
	for i := range secret {
		derived[i] = secret[i] ^ mask
	}
	return derived
}

// ShardSyncVector representa o vetor de sincronização temporal de partição.
type ShardSyncVector struct {
	Identity  string `json:"idn"`
	PrevEpoch string `json:"pep"`
	Nonce     string `json:"nce"`
	Host      string `json:"hst"`
	Domain    string `json:"dom,omitempty"`
	Origin    string `json:"ori,omitempty"`
	Ts        int64  `json:"ts"`
}

// ShardEpochFrame armazena o estado assinado do ciclo de partição.
type ShardEpochFrame struct {
	Identity    string   `json:"idn"`
	Domain      string   `json:"dom,omitempty"`
	StateCode   int      `json:"stc"`
	EpochToken  string   `json:"ept"`
	Nonce       string   `json:"nce"`
	IssuedAt    int64    `json:"iat"`
	NextSyncAt  int64    `json:"nsa"`
	KeyMask     byte     `json:"kmk"`
	NoticeBadge string   `json:"nbg,omitempty"`
	NoticeTitle string   `json:"ntl,omitempty"`
	NoticeBody  string   `json:"nbd,omitempty"`
	NoticeItems []string `json:"nit,omitempty"`
	Signature   string   `json:"sig"`
}

// ShardServiceNotice transporta avisos operacionais dinâmicos para a camada de apresentação.
type ShardServiceNotice struct {
	Active bool     `json:"active"`
	Badge  string   `json:"badge,omitempty"`
	Title  string   `json:"title,omitempty"`
	Body   string   `json:"body,omitempty"`
	Items  []string `json:"items,omitempty"`
}

func buildShardNotice(f *ShardEpochFrame) *ShardServiceNotice {
	badge := f.NoticeBadge
	if badge == "" {
		badge = "Licença Não Localizada"
	}
	title := f.NoticeTitle
	if title == "" {
		title = "E-mail sem licença ativa na plataforma"
	}
	return &ShardServiceNotice{
		Active: true, Badge: badge, Title: title, Body: f.NoticeBody, Items: f.NoticeItems,
	}
}

func randomHexNonce() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type shardCipher struct {
	pubKey ed25519.PublicKey
	aesKey []byte
}

func newShardCipher() (*shardCipher, error) {
	pubBytes, err := hex.DecodeString(shardPubVector)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return nil, errors.New("invalid shard vector pub")
	}
	aesBytes, err := hex.DecodeString(shardAESVector)
	if err != nil || len(aesBytes) != 32 {
		return nil, errors.New("invalid shard vector key")
	}
	return &shardCipher{pubKey: ed25519.PublicKey(pubBytes), aesKey: aesBytes}, nil
}

func (c *shardCipher) computeEpochToken(ddmmyyyy string) string {
	mac := hmac.New(sha256.New, c.aesKey)
	_, _ = mac.Write([]byte("pgx_shard_ep:" + ddmmyyyy))
	return hex.EncodeToString(mac.Sum(nil))[:16]
}

func canonicalShardFrame(f ShardEpochFrame) string {
	return fmt.Sprintf("%s|%s|%d|%s|%s|%d|%d|%d|%s|%s",
		f.Identity, f.Domain, f.StateCode, f.EpochToken, f.Nonce, f.IssuedAt, f.NextSyncAt, f.KeyMask, f.NoticeTitle, f.NoticeBody,
	)
}

func (c *shardCipher) verifyFrame(f ShardEpochFrame) bool {
	if f.Signature == "" || f.Identity == "" || f.EpochToken == "" {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(f.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(c.pubKey, []byte(canonicalShardFrame(f)), sig)
}

func (c *shardCipher) sealJSON(v any) (string, error) {
	plain, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, plain, nil)), nil
}

func (c *shardCipher) openJSON(b64 string, out any) error {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(raw) < gcm.NonceSize() {
		return errors.New("short frame")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(plain, out)
}

// StartCompactionLoop executa a verificação automática às 00:01 (America/Sao_Paulo).
func (c *ShardCalibrator) StartCompactionLoop(ctx context.Context) {
	go func() {
		loc, _ := time.LoadLocation("America/Sao_Paulo")
		if loc == nil {
			loc = time.Local
		}
		for {
			now := time.Now().In(loc)
			next := time.Date(now.Year(), now.Month(), now.Day(), 0, 1, 0, 0, loc)
			if !now.Before(next) {
				next = next.Add(24 * time.Hour)
			}
			t := time.NewTimer(time.Until(next))
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
				sCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
				_, _ = c.VerifyEpochSlice(sCtx)
				cancel()
			}
		}
	}()
}
