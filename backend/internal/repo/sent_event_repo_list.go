package repo

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// ListFilter agrupa os filtros aceitos por List.
type ListFilter struct {
	AccountID   int64     // 0 = todos (owner); >0 = filtra por conta (member)
	PixelName   string    // exato (vazio = todos)
	EventNames  []string  // event_name IN (...) — vazio = todos
	ExternalID  string    // exato (vazio = todos) — histórico de compras de 1 cliente
	ExternalIDs []string  // external_id = ANY(...) — múltiplos external_ids da mesma pessoa/grupo
	Status      string    // queued|sent|failed (vazio = todos)
	From        time.Time // created_at >= From (zero = sem filtro)
	To          time.Time // created_at < To (zero = sem filtro)
	Limit       int       // default 50, max 500
	Offset      int
	IncludeLTV         bool
	IncludeBuyer       bool
	IncludeUserContext bool
	TrafficSource      string
}

// ListGrouped retorna a lista agregada por (event_id, pixel_name): uma
// linha por evento, com array de canais que receberam aquele envio.
//
// Estrategia: faz a paginacao em uma subquery selecionando event_ids
// distintos ordenados pelo created_at mais recente, e depois faz JOIN
// pra puxar todos os canais de cada event_id da pagina.
func (r *SentEventRepo) ListGrouped(ctx context.Context, f ListFilter) ([]GroupedEvent, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 500 {
		f.Limit = 500
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	args := make([]any, 0, 6)
	conds := make([]string, 0, 6)
	add := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, strings.Replace(cond, "$N", "$"+itoa(len(args)), 1))
	}
	if f.AccountID != 0 {
		add("account_id = $N", f.AccountID)
	}
	if f.PixelName != "" {
		add("pixel_name = $N", f.PixelName)
	}
	if len(f.EventNames) > 0 {
		add("event_name = ANY($N::text[])", f.EventNames)
	}
	if len(f.ExternalIDs) > 0 {
		add("external_id = ANY($N::text[])", f.ExternalIDs)
	} else if f.ExternalID != "" {
		add("external_id = $N", f.ExternalID)
	}
	if f.Status != "" {
		add("status = $N", f.Status)
	} else {
		conds = append(conds, "status <> 'contested'")
	}
	if !f.From.IsZero() {
		add("created_at >= $N", f.From)
	}
	if !f.To.IsZero() {
		add("created_at < $N", f.To)
	}
	if srcCond := trafficSourceUserCond(f.TrafficSource, "u."); srcCond != "" {
		conds = append(conds, "EXISTS (SELECT 1 FROM tracked_users u WHERE u.external_id = sent_events.external_id AND u.account_id = sent_events.account_id AND "+srcCond+")")
	}
	conds = append(conds, excludeHighVolumeEvents)
	where := "WHERE " + strings.Join(conds, " AND ")

	// Total de eventos distintos (event_id + pixel_name).
	var total int64
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM (SELECT DISTINCT event_id, pixel_name FROM sent_events "+where+") t",
		args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Pega event_ids da pagina (mais recentes primeiro).
	pageArgs := append([]any{}, args...)
	pageArgs = append(pageArgs, f.Limit, f.Offset)
	pageQ := "SELECT event_id, pixel_name, MAX(created_at) AS last_at FROM sent_events " +
		where + " GROUP BY event_id, pixel_name ORDER BY last_at DESC, event_id DESC " +
		"LIMIT $" + itoa(len(args)+1) + " OFFSET $" + itoa(len(args)+2)

	rows, err := r.pool.Query(ctx, pageQ, pageArgs...)
	if err != nil {
		return nil, 0, err
	}
	type key struct {
		eventID, pixelName string
	}
	keys := make([]key, 0, f.Limit)
	keysSet := make(map[key]int) // index na slice de keys
	for rows.Next() {
		var k key
		var lastAt time.Time
		if err := rows.Scan(&k.eventID, &k.pixelName, &lastAt); err != nil {
			rows.Close()
			return nil, 0, err
		}
		keysSet[k] = len(keys)
		keys = append(keys, k)
	}
	rows.Close()
	if len(keys) == 0 {
		return []GroupedEvent{}, total, nil
	}

	// Busca todas as linhas dos event_ids da pagina.
	eventIDs := make([]string, len(keys))
	pixelNames := make([]string, len(keys))
	for i, k := range keys {
		eventIDs[i] = k.eventID
		pixelNames[i] = k.pixelName
	}
	detailQ := `SELECT id, event_id, event_name, pixel_name, external_id, channel, status,
		response_code, error, purchase_value, created_at, COALESCE(aggregated_count, 1), COALESCE(aggregated_items::text, ''),
		COALESCE(input_json->>'first_name', ''), COALESCE(input_json->>'last_name', ''), COALESCE(input_json->>'phone', '')
		FROM sent_events
		WHERE event_id = ANY($1::text[]) AND pixel_name = ANY($2::text[])
		  AND ($3::bigint = 0 OR account_id = $3) AND ($4::text <> '' OR status <> 'contested')
		ORDER BY id DESC`
	drows, err := r.pool.Query(ctx, detailQ, eventIDs, pixelNames, f.AccountID, f.Status)
	if err != nil {
		return nil, 0, err
	}
	defer drows.Close()

	// Constroi GroupedEvent na ordem da pagina (preservando keys order).
	out := make([]GroupedEvent, len(keys))
	for i, k := range keys {
		out[i] = GroupedEvent{
			EventID:   k.eventID,
			PixelName: k.pixelName,
			Channels:  []ChannelBrief{},
		}
	}

	for drows.Next() {
		var (
			id            int64
			eventID       string
			eventName     string
			pixelName     string
			externalID    string
			channel       string
			status        string
			responseCode  int
			errMsg        string
			purchaseValue float64
			createdAt     time.Time
			aggCount      int
			aggItemsJSON  string
			buyerFirst    string
			buyerLast     string
			buyerPhone    string
		)
		if err := drows.Scan(&id, &eventID, &eventName, &pixelName, &externalID,
			&channel, &status, &responseCode, &errMsg, &purchaseValue, &createdAt,
			&aggCount, &aggItemsJSON, &buyerFirst, &buyerLast, &buyerPhone); err != nil {
			return nil, 0, err
		}
		idx, ok := keysSet[key{eventID, pixelName}]
		if !ok {
			continue
		}
		if out[idx].EventName == "" {
			out[idx].EventName = eventName
			out[idx].ExternalID = externalID
			out[idx].CreatedAt = createdAt
		}
		if f.IncludeBuyer {
			if out[idx].BuyerFirstName == "" && buyerFirst != "" {
				out[idx].BuyerFirstName = strings.TrimSpace(buyerFirst)
			}
			if out[idx].BuyerLastName == "" && buyerLast != "" {
				out[idx].BuyerLastName = strings.TrimSpace(buyerLast)
			}
			if out[idx].BuyerPhone == "" && buyerPhone != "" {
				out[idx].BuyerPhone = strings.TrimSpace(buyerPhone)
			}
		}
		if purchaseValue > out[idx].PurchaseValue {
			out[idx].PurchaseValue = purchaseValue
		}
		if aggCount > out[idx].AggregatedCount {
			out[idx].AggregatedCount = aggCount
		}
		if len(out[idx].AggregatedItems) == 0 && aggItemsJSON != "" && aggItemsJSON != "null" {
			var items []domain.AggregatedItem
			if json.Unmarshal([]byte(aggItemsJSON), &items) == nil {
				out[idx].AggregatedItems = items
			}
		}
		channelExists := false
		for _, ch := range out[idx].Channels {
			if ch.Channel == channel {
				channelExists = true
				break
			}
		}
		if !channelExists {
			out[idx].Channels = append(out[idx].Channels, ChannelBrief{
				ID:           id,
				Channel:      channel,
				Status:       status,
				ResponseCode: responseCode,
				HasError:     errMsg != "" || status == "failed",
			})
		}
	}
	if err := drows.Err(); err != nil {
		return nil, 0, err
	}

	if f.IncludeLTV {
		if err := r.populateLTV(ctx, f.AccountID, out); err != nil {
			return nil, 0, err
		}
	}
	if f.IncludeBuyer {
		if err := r.populateBuyerInfo(ctx, f.AccountID, out); err != nil {
			return nil, 0, err
		}
	}
	if f.IncludeUserContext {
		if err := r.populateUserContext(ctx, f.AccountID, out); err != nil {
			return nil, 0, err
		}
	}

	return out, total, nil
}
