package domain

import "time"

// EventCounter é o contador agregado de um par (pixel, evento) numa janela de
// 1 hora. Usado em vez de persistir cada PageView/ViewContent/scroll inteiro
// na tabela sent_events.
type EventCounter struct {
	ID           int64     `json:"id"`
	PixelName    string    `json:"pixel_name"`
	EventName    string    `json:"event_name"`
	BucketAt     time.Time `json:"bucket_at"`
	CountTotal   int64     `json:"count_total"`
	CountSent    int64     `json:"count_sent"`
	CountFailed  int64     `json:"count_failed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CounterDelta é a unidade de incremento aplicada no UPSERT. Cada evento
// processado pelo worker gera um delta com count_total=1 e (sent ou failed)=1
// dependendo do resultado.
type CounterDelta struct {
	Total  int64
	Sent   int64
	Failed int64
}

// eventsWithoutFullPayload são os únicos eventos que NÃO justificam payload
// completo em sent_events — ficam só no agregado de event_counters. São de
// altíssimo volume e sem valor de auditoria individual (a página de Eventos
// não precisa listar cada PageView/ViewContent um a um).
//
// Todo o resto (leads, vendas, funil de checkout, wishlist, scroll/timer/
// vídeo, business messaging) é persistido com payload completo e aparece na
// página de Eventos.
var eventsWithoutFullPayload = map[string]bool{
	EventPageView:    true,
	EventViewContent: true,
}

// ShouldStoreFullPayload retorna true se o evento deve ser persistido em
// sent_events com payload completo (e portanto aparecer na página de
// Eventos). Hoje só PageView e ViewContent ficam de fora.
func ShouldStoreFullPayload(eventName string) bool {
	return !eventsWithoutFullPayload[eventName]
}
