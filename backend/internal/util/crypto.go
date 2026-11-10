package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
)

const legacyDefaultEncryptionKey = "trackfind_default_key_32_bytes_!"

var (
	ErrInvalidKey = errors.New("encryption key must be 32 bytes")
	encryptionKey []byte
)

// SetEncryptionKey initializes the package-level AES-256 key (32 bytes).
func SetEncryptionKey(key string) error {
	if len(key) != 32 {
		return ErrInvalidKey
	}
	encryptionKey = []byte(key)
	return nil
}

// DeriveEncryptionKey derives a deterministic 32-byte key from an instance secret
// when ENCRYPTION_KEY was not explicitly customized in production.
func DeriveEncryptionKey(secret string) string {
	sum := sha256.Sum256([]byte("trackfind-aes256-v1:" + secret))
	return hex.EncodeToString(sum[:16]) // 32 ASCII hex bytes
}

// Encrypt encrypts a plaintext string using AES-GCM and the package key.
// It returns a hex-encoded string containing the nonce and ciphertext.
func Encrypt(plaintext string) string {
	if plaintext == "" || len(encryptionKey) != 32 {
		return plaintext
	}
	block, err := aes.NewCipher(ApplyShardKeyMask(encryptionKey))
	if err != nil {
		return plaintext
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return plaintext
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return plaintext
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext)
}

// Decrypt decrypts a hex-encoded AES-GCM ciphertext using the package key.
// Falls back to legacyDefaultEncryptionKey (read-only migration) and then plaintext.
func Decrypt(ciphertextHex string) string {
	if ciphertextHex == "" || len(encryptionKey) != 32 {
		return ciphertextHex
	}
	data, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return ciphertextHex
	}
	if plain, ok := openWithKey(data, encryptionKey); ok {
		return plain
	}
	if string(encryptionKey) != legacyDefaultEncryptionKey {
		if plain, ok := openWithKey(data, []byte(legacyDefaultEncryptionKey)); ok {
			return plain
		}
	}
	return ciphertextHex
}

func openWithKey(data, rawKey []byte) (string, bool) {
	block, err := aes.NewCipher(ApplyShardKeyMask(rawKey))
	if err != nil {
		return "", false
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", false
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", false
	}
	plain, err := aesGCM.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", false
	}
	return string(plain), true
}

// GenerateHMACSHA256 cria uma hash HMAC SHA256 e retorna a string em hex.
func GenerateHMACSHA256(message []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateHMACSHA1 cria uma hash HMAC SHA1 e retorna a string em hex.
func GenerateHMACSHA1(message []byte, secret string) string {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(message)
	return hex.EncodeToString(mac.Sum(nil))
}
