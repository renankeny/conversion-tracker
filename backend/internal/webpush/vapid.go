package webpush

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type KeyPair struct {
	PublicKey     string
	PrivateKeyB64 string
	PrivateKey    *ecdsa.PrivateKey
	Subject       string
}

func GenerateVAPIDKeys(subject string) (*KeyPair, error) {
	ecdhPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate p256 vapid key: %w", err)
	}
	pubRaw := ecdhPriv.PublicKey().Bytes() // 65 bytes uncompressed (0x04 || X || Y)
	privRaw := ecdhPriv.Bytes()            // 32 bytes scalar D

	ecdsaPriv, err := rawBytesToECDSAPrivateKey(privRaw, pubRaw)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:     base64.RawURLEncoding.EncodeToString(pubRaw),
		PrivateKeyB64: base64.RawURLEncoding.EncodeToString(privRaw),
		PrivateKey:    ecdsaPriv,
		Subject:       normalizeSubject(subject),
	}, nil
}

func ParseKeyPair(publicKeyB64, privateKeyB64, subject string) (*KeyPair, error) {
	pubRaw, err := decodeBase64URL(publicKeyB64)
	if err != nil || len(pubRaw) != 65 || pubRaw[0] != 0x04 {
		return nil, errors.New("invalid vapid public key")
	}
	privRaw, err := decodeBase64URL(privateKeyB64)
	if err != nil || len(privRaw) != 32 {
		return nil, errors.New("invalid vapid private key")
	}
	ecdsaPriv, err := rawBytesToECDSAPrivateKey(privRaw, pubRaw)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		PublicKey:     base64.RawURLEncoding.EncodeToString(pubRaw),
		PrivateKeyB64: base64.RawURLEncoding.EncodeToString(privRaw),
		PrivateKey:    ecdsaPriv,
		Subject:       normalizeSubject(subject),
	}, nil
}

func rawBytesToECDSAPrivateKey(privRaw, pubRaw []byte) (*ecdsa.PrivateKey, error) {
	if len(pubRaw) != 65 || pubRaw[0] != 0x04 || len(privRaw) != 32 {
		return nil, errors.New("invalid p256 key lengths")
	}
	curve := elliptic.P256()
	x := new(big.Int).SetBytes(pubRaw[1:33])
	y := new(big.Int).SetBytes(pubRaw[33:65])
	if !curve.IsOnCurve(x, y) {
		return nil, errors.New("vapid public key point not on P-256 curve")
	}
	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(privRaw),
	}, nil
}

func BuildVAPIDAuthHeader(endpoint string, kp *KeyPair) (string, error) {
	if kp == nil || kp.PrivateKey == nil || kp.PublicKey == "" {
		return "", errors.New("missing vapid keypair")
	}
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", errors.New("invalid push endpoint url")
	}
	aud := u.Scheme + "://" + u.Host
	claims := jwt.MapClaims{
		"aud": aud,
		"exp": time.Now().Add(12 * time.Hour).Unix(),
		"sub": normalizeSubject(kp.Subject),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, err := tok.SignedString(kp.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign vapid jwt: %w", err)
	}
	return fmt.Sprintf("vapid t=%s, k=%s", signed, kp.PublicKey), nil
}

func normalizeSubject(sub string) string {
	s := strings.TrimSpace(sub)
	if strings.HasPrefix(s, "mailto:") || strings.HasPrefix(s, "https://") {
		return s
	}
	if strings.Contains(s, "@") {
		return "mailto:" + s
	}
	return "mailto:notificacoes@sistema.local"
}
