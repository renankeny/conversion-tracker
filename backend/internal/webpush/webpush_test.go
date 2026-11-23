package webpush

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVAPIDGenerateParseAndAuthHeader(t *testing.T) {
	kp, err := GenerateVAPIDKeys("admin@trackeado.com.br")
	if err != nil {
		t.Fatalf("GenerateVAPIDKeys failed: %v", err)
	}
	parsed, err := ParseKeyPair(kp.PublicKey, kp.PrivateKeyB64, kp.Subject)
	if err != nil {
		t.Fatalf("ParseKeyPair failed: %v", err)
	}
	if parsed.PublicKey != kp.PublicKey || parsed.Subject != "mailto:admin@trackeado.com.br" {
		t.Fatalf("unexpected parsed keypair: %+v", parsed)
	}
	hdr, err := BuildVAPIDAuthHeader("https://fcm.googleapis.com/fcm/send/abc123", parsed)
	if err != nil {
		t.Fatalf("BuildVAPIDAuthHeader failed: %v", err)
	}
	if !strings.HasPrefix(hdr, "vapid t=") || !strings.Contains(hdr, ", k="+kp.PublicKey) {
		t.Fatalf("invalid VAPID header format: %s", hdr)
	}
}

func TestEncryptAES128GCMRoundTripAndSendGone(t *testing.T) {
	uaPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	authSecret := make([]byte, 16)
	_, _ = rand.Read(authSecret)

	p256dhB64 := base64.RawURLEncoding.EncodeToString(uaPriv.PublicKey().Bytes())
	authB64 := base64.RawURLEncoding.EncodeToString(authSecret)
	msg := []byte(`{"title":"Nova Venda","body":"R$ 497,00 aprovada"}`)

	enc, err := EncryptAES128GCM(msg, p256dhB64, authB64)
	if err != nil {
		t.Fatalf("EncryptAES128GCM failed: %v", err)
	}
	if len(enc) <= 86 {
		t.Fatalf("ciphertext too short: %d", len(enc))
	}
	salt := enc[0:16]
	rs := binary.BigEndian.Uint32(enc[16:20])
	idLen := int(enc[20])
	if rs != 4096 || idLen != 65 {
		t.Fatalf("unexpected header rs=%d idLen=%d", rs, idLen)
	}
	asPubBytes := enc[21 : 21+idLen]
	asPub, err := ecdh.P256().NewPublicKey(asPubBytes)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := uaPriv.ECDH(asPub)
	if err != nil {
		t.Fatal(err)
	}
	keyInfo := append([]byte("WebPush: info\x00"), uaPriv.PublicKey().Bytes()...)
	keyInfo = append(keyInfo, asPubBytes...)
	ikm, _ := deriveHKDF(shared, authSecret, keyInfo, 32)
	cek, _ := deriveHKDF(ikm, salt, []byte("Content-Encoding: aes128gcm\x00"), 16)
	nonce, _ := deriveHKDF(ikm, salt, []byte("Content-Encoding: nonce\x00"), 12)
	block, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, enc[86:], nil)
	if err != nil {
		t.Fatalf("subscriber decrypt failed: %v", err)
	}
	if plain[len(plain)-1] != 0x02 || !bytes.Equal(plain[:len(plain)-1], msg) {
		t.Fatalf("decrypted payload mismatch: %q", string(plain))
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer srv.Close()

	kp, _ := GenerateVAPIDKeys("mailto:test@trackeado.com.br")
	gone, sendErr := NewSender().Send(context.Background(), kp, SubscriptionTarget{
		Endpoint:   strings.Replace(srv.URL, "127.0.0.1", "localhost", 1) + "/push/sub1",
		P256dh:     p256dhB64,
		AuthSecret: authB64,
	}, msg)
	if !gone || sendErr == nil {
		t.Fatalf("expected 410 Gone detection, got gone=%v err=%v", gone, sendErr)
	}
}
