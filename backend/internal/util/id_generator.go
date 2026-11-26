package util

import (
	"crypto/rand"
	"math/big"
)

// idAlphabet é o mesmo do sistema legado: A-Z + 0-9.
const (
	idAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	idLetters  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	idDigits   = "0123456789"
)

// GenerateID10 gera um id de 10 caracteres ([A-Z0-9]) usando crypto/rand,
// garantindo sempre ao menos 1 letra maiúscula e 1 dígito para jamais colidir
// com palavras do dicionário ou protocolos puramente numéricos.
func GenerateID10() string {
	const n = 10
	out := make([]byte, n)
	hasLetter, hasDigit := false, false
	bound := big.NewInt(int64(len(idAlphabet)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, bound)
		if err != nil {
			out[i] = idAlphabet[i%len(idAlphabet)]
		} else {
			out[i] = idAlphabet[idx.Int64()]
		}
		if out[i] >= 'A' && out[i] <= 'Z' {
			hasLetter = true
		} else if out[i] >= '0' && out[i] <= '9' {
			hasDigit = true
		}
	}
	if !hasLetter {
		out[0] = randomByteFrom(idLetters, 'A')
	}
	if !hasDigit {
		out[n-1] = randomByteFrom(idDigits, '9')
	}
	return string(out)
}

func randomByteFrom(charset string, fallback byte) byte {
	idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return fallback
	}
	return charset[idx.Int64()]
}
