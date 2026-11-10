package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func computeMediaMAC(secret string, accountID, messageID, expUnix int64) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "wamedia:v1:%d:%d:%d", accountID, messageID, expUnix)
	return mac.Sum(nil)
}

// SignMediaURL gera uma URL pública temporária assinada com HMAC-SHA256 que permite
// aos elementos <audio>, <img> e links de documentos fazerem streaming direto (incluindo
// HTTP Range 206 Partial Content no Safari/Chrome) sem precisar expor o MinIO publicamente.
func SignMediaURL(publicAPIURL, secret string, accountID, messageID int64, ttl time.Duration) string {
	if ttl <= 0 {
		ttl = 2 * time.Hour
	}
	exp := time.Now().Add(ttl).Unix()
	sig := hex.EncodeToString(computeMediaMAC(secret, accountID, messageID, exp))
	base := strings.TrimRight(strings.TrimSpace(publicAPIURL), "/")
	return fmt.Sprintf("%s/media/whatsapp/%d?acc=%d&exp=%d&sig=%s", base, messageID, accountID, exp, sig)
}

func VerifyMediaSignature(secret string, accountID, messageID, expUnix int64, sigHex string, now time.Time) bool {
	if secret == "" || accountID <= 0 || messageID <= 0 || expUnix <= 0 {
		return false
	}
	if now.Unix() > expUnix {
		return false
	}
	got, err := hex.DecodeString(strings.TrimSpace(sigHex))
	if err != nil || len(got) != sha256.Size {
		return false
	}
	expected := computeMediaMAC(secret, accountID, messageID, expUnix)
	return hmac.Equal(got, expected)
}

