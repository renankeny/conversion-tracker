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

// FullPayloadEvents é a lista de eventos que justificam guardar payload
// completo em sent_events. Os demais entram apenas no event_counters.
//
// A lista é fixa pra ficar previsível — pode ser tornada configurável via env
// se algum cliente precisar.
var FullPayloadEvents = map[string]bool{
	EventLead:                 true,
	EventInitiateCheckout:     true,
	EventPurchase:             true,
	EventAddPaymentInfo:       true,
	EventCompleteRegistration: true,
	EventContact:              true,
	EventAddToCart:            true, // valioso pra funil de e-commerce
}

// ShouldStoreFullPayload retorna true se o evento deve ser persistido em
// sent_events com payload completo. Os demais vão apenas para event_counters.
func ShouldStoreFullPayload(eventName string) bool {
	return FullPayloadEvents[eventName]
}
