package repo

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

// ListGrouped retorna a lista agregada por (event_id, pixel_name): uma
// linha por evento com o array de canais disparados.
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

	where, args := buildGroupedWhere(f)

	var total int64
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM (SELECT DISTINCT event_id, pixel_name FROM sent_events "+where+") t",
		args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

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
	keysSet := make(map[key]int)
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

	eventIDs := make([]string, len(keys))
	pixelNames := make([]string, len(keys))
	for i, k := range keys {
		eventIDs[i] = k.eventID
		pixelNames[i] = k.pixelName
	}

	wantContested := f.Status == "contested"
	detailQ := `SELECT se.id, se.event_id, se.event_name, se.pixel_name, se.external_id, se.channel, se.status,
		se.response_code, se.error, se.purchase_value, se.created_at,
		COALESCE(se.aggregated_count, 1), COALESCE(se.aggregated_items::text, ''),
		CASE WHEN $5::boolean THEN COALESCE(se.input_json->>'first_name', '') ELSE '' END,
		CASE WHEN $5::boolean THEN COALESCE(se.input_json->>'last_name', '') ELSE '' END,
		CASE WHEN $5::boolean THEN COALESCE(se.input_json->>'phone', '') ELSE '' END
		FROM sent_events se
		JOIN UNNEST($1::text[], $2::text[]) AS p(event_id, pixel_name)
		  ON se.event_id = p.event_id AND se.pixel_name = p.pixel_name
		WHERE ($3::bigint = 0 OR se.account_id = $3)
		  AND (CASE WHEN $4::boolean THEN se.status = 'contested' ELSE se.status <> 'contested' END)
		ORDER BY se.id DESC`
	drows, err := r.pool.Query(ctx, detailQ, eventIDs, pixelNames, f.AccountID, wantContested, f.IncludeBuyer)
	if err != nil {
		return nil, 0, err
	}
	defer drows.Close()

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
			id, aggCount                                         int64
			responseCode                                         int
			eventID, eventName, pixelName, externalID            string
			channel, status, errMsg, aggItemsJSON                string
			buyerFirst, buyerLast, buyerPhone                    string
			purchaseValue                                        float64
			createdAt                                            time.Time
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
		if int(aggCount) > out[idx].AggregatedCount {
			out[idx].AggregatedCount = int(aggCount)
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
