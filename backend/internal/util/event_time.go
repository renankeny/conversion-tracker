package util

import "time"

// ClampEventTime normaliza event_time recebido de integrações externas:
//   - converte milissegundos para segundos;
//   - limita a janela aceita pela Meta CAPI: [now-7d, now+60s].
//
// Valores fora da janela são trazidos para o limite mais próximo em vez de gerar 400 permanente.
func ClampEventTime(ts int64, now time.Time) int64 {
	if ts > 1e12 { // milissegundos
		ts /= 1000
	}
	minTs := now.Add(-7 * 24 * time.Hour).Add(time.Minute).Unix()
	maxTs := now.Add(60 * time.Second).Unix()
	if ts < minTs {
		return minTs
	}
	if ts > maxTs {
		return now.Unix()
	}
	return ts
}
