package util

import (
	"crypto/rand"
	"math/big"
)

// idAlphabet é o mesmo do sistema legado: A-Z + 0-9.
const idAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateID10 gera um id de 10 caracteres ([A-Z0-9]) usando crypto/rand.
// Compatível com o id gerado pelo pixel.js no client.
func GenerateID10() string {
	const n = 10
	out := make([]byte, n)
	bound := big.NewInt(int64(len(idAlphabet)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, bound)
		if err != nil {
			// rand.Reader nunca falha em ambiente normal; em caso extremo
			// devolve 'A' pra não quebrar a chamada.
			out[i] = idAlphabet[0]
			continue
		}
		out[i] = idAlphabet[idx.Int64()]
	}
	return string(out)
}
