package webpush

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
)

const defaultRecordSize uint32 = 4096

// EncryptAES128GCM implementa RFC 8291 + RFC 8188 (Content-Encoding: aes128gcm)
// para entrega universal em Chrome, Edge, Firefox, Android e Safari (macOS/iOS 16.4+).
func EncryptAES128GCM(plaintext []byte, p256dhB64, authB64 string) ([]byte, error) {
	uaPubBytes, err := decodeBase64URL(p256dhB64)
	if err != nil || len(uaPubBytes) != 65 || uaPubBytes[0] != 0x04 {
		return nil, errors.New("invalid subscriber p256dh public key")
	}
	authSecret, err := decodeBase64URL(authB64)
	if err != nil || len(authSecret) < 12 {
		return nil, errors.New("invalid subscriber auth secret")
	}

	curve := ecdh.P256()
	uaPub, err := curve.NewPublicKey(uaPubBytes)
	if err != nil {
		return nil, fmt.Errorf("parse subscriber public key: %w", err)
	}

	asPriv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral key: %w", err)
	}
	asPubBytes := asPriv.PublicKey().Bytes()

	ecdhSecret, err := asPriv.ECDH(uaPub)
	if err != nil {
		return nil, fmt.Errorf("ecdh shared secret: %w", err)
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	// key_info = "WebPush: info\x00" || ua_public || as_public
	keyInfo := make([]byte, 0, 14+len(uaPubBytes)+len(asPubBytes))
	keyInfo = append(keyInfo, []byte("WebPush: info\x00")...)
	keyInfo = append(keyInfo, uaPubBytes...)
	keyInfo = append(keyInfo, asPubBytes...)

	ikm, err := deriveHKDF(ecdhSecret, authSecret, keyInfo, 32)
	if err != nil {
		return nil, err
	}

	cek, err := deriveHKDF(ikm, salt, []byte("Content-Encoding: aes128gcm\x00"), 16)
	if err != nil {
		return nil, err
	}
	nonce, err := deriveHKDF(ikm, salt, []byte("Content-Encoding: nonce\x00"), 12)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// RFC 8188 final record delimiter byte 0x02
	padded := make([]byte, len(plaintext)+1)
	copy(padded, plaintext)
	padded[len(plaintext)] = 0x02

	ciphertext := gcm.Seal(nil, nonce, padded, nil)

	// Header: salt (16) || rs (4) || idlen (1) || keyid (65) = 86 bytes
	header := make([]byte, 16+4+1+len(asPubBytes))
	copy(header[0:16], salt)
	binary.BigEndian.PutUint32(header[16:20], defaultRecordSize)
	header[20] = byte(len(asPubBytes))
	copy(header[21:], asPubBytes)

	out := make([]byte, len(header)+len(ciphertext))
	copy(out, header)
	copy(out[len(header):], ciphertext)
	return out, nil
}

func deriveHKDF(secret, salt, info []byte, length int) ([]byte, error) {
	r := hkdf.New(sha256.New, secret, salt, info)
	out := make([]byte, length)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, fmt.Errorf("hkdf expand: %w", err)
	}
	return out, nil
}

func decodeBase64URL(s string) ([]byte, error) {
	clean := strings.TrimSpace(s)
	clean = strings.TrimRight(clean, "=")
	clean = strings.ReplaceAll(clean, "+", "-")
	clean = strings.ReplaceAll(clean, "/", "_")
	return base64.RawURLEncoding.DecodeString(clean)
}
