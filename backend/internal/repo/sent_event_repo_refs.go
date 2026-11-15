package repo

import (
	"context"
	"strings"
)

// EventRef referencia um evento especificando event_id e, opcionalmente, pixel_name.
type EventRef struct {
	EventID   string `json:"event_id"`
	PixelName string `json:"pixel_name"`
}

// DeletedEventTuple identifica cada evento efetivamente removido no banco,
// permitindo limpar travas de deduplicação no Redis com precisão.
type DeletedEventTuple struct {
	AccountID int64
	PixelName string
	EventID   string
}

func splitEventRefs(refs []EventRef) (pairIDs, pairPixels, anyPixelIDs []string, normalized []EventRef) {
	pairIDs = make([]string, 0, len(refs))
	pairPixels = make([]string, 0, len(refs))
	anyPixelIDs = make([]string, 0, len(refs))
	normalized = make([]EventRef, 0, len(refs))
	seen := make(map[EventRef]bool, len(refs))

	for _, r := range refs {
		evID := strings.TrimSpace(r.EventID)
		px := strings.TrimSpace(r.PixelName)
		if evID == "" {
			continue
		}
		clean := EventRef{EventID: evID, PixelName: px}
		if seen[clean] {
			continue
		}
		seen[clean] = true
		normalized = append(normalized, clean)
		if px != "" {
			pairIDs = append(pairIDs, evID)
			pairPixels = append(pairPixels, px)
		} else {
			anyPixelIDs = append(anyPixelIDs, evID)
		}
	}
	return pairIDs, pairPixels, anyPixelIDs, normalized
}

// DeleteByRefs remove eventos correspondentes aos pares (event_id, pixel_name)
// ou apenas event_id (fallback legado quando pixel_name estiver vazio),
// respeitando estritamente o isolamento multi-tenant.
func (r *SentEventRepo) DeleteByRefs(ctx context.Context, accountID int64, refs []EventRef) (int64, []DeletedEventTuple, error) {
	pairIDs, pairPixels, anyPixelIDs, norm := splitEventRefs(refs)
	if len(norm) == 0 {
		return 0, nil, nil
	}

	const q = `
		DELETE FROM sent_events se
		WHERE ($1::bigint = 0 OR se.account_id = $1)
		  AND (
		    EXISTS (
		      SELECT 1 FROM UNNEST($2::text[], $3::text[]) AS p(event_id, pixel_name)
		      WHERE se.event_id = p.event_id AND se.pixel_name = p.pixel_name
		    )
		    OR se.event_id = ANY($4::text[])
		  )
		RETURNING se.account_id, se.pixel_name, se.event_id
	`
	rows, err := r.pool.Query(ctx, q, accountID, pairIDs, pairPixels, anyPixelIDs)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var deletedCount int64
	tuples := make([]DeletedEventTuple, 0, len(norm))
	seenTuple := make(map[DeletedEventTuple]bool, len(norm))
	for rows.Next() {
		var t DeletedEventTuple
		if err := rows.Scan(&t.AccountID, &t.PixelName, &t.EventID); err != nil {
			return deletedCount, nil, err
		}
		deletedCount++
		if !seenTuple[t] {
			seenTuple[t] = true
			tuples = append(tuples, t)
		}
	}
	if err := rows.Err(); err != nil {
		return deletedCount, nil, err
	}
	return deletedCount, tuples, nil
}

// FindInputsByRefs busca o input_json original dos eventos referenciados,
// suportando correspondência exata por (event_id, pixel_name) e fallback por event_id.
func (r *SentEventRepo) FindInputsByRefs(ctx context.Context, accountID int64, refs []EventRef) ([][]byte, int, error) {
	pairIDs, pairPixels, anyPixelIDs, norm := splitEventRefs(refs)
	if len(norm) == 0 {
		return nil, 0, nil
	}

	const q = `
		SELECT DISTINCT ON (se.event_id, se.pixel_name)
			se.event_id, se.pixel_name, se.input_json::text
		FROM sent_events se
		WHERE se.input_json IS NOT NULL
		  AND ($1::bigint = 0 OR se.account_id = $1)
		  AND (
		    EXISTS (
		      SELECT 1 FROM UNNEST($2::text[], $3::text[]) AS p(event_id, pixel_name)
		      WHERE se.event_id = p.event_id AND se.pixel_name = p.pixel_name
		    )
		    OR se.event_id = ANY($4::text[])
		  )
		ORDER BY se.event_id, se.pixel_name, se.id DESC
	`
	rows, err := r.pool.Query(ctx, q, accountID, pairIDs, pairPixels, anyPixelIDs)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	byPair := make(map[EventRef][]byte, len(norm))
	byEventID := make(map[string][]byte, len(norm))
	for rows.Next() {
		var evID, px string
		var raw []byte
		if err := rows.Scan(&evID, &px, &raw); err != nil {
			return nil, 0, err
		}
		byPair[EventRef{EventID: evID, PixelName: px}] = raw
		if _, exists := byEventID[evID]; !exists {
			byEventID[evID] = raw
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	inputs := make([][]byte, 0, len(norm))
	notFound := 0
	for _, ref := range norm {
		if ref.PixelName != "" {
			if raw, ok := byPair[ref]; ok {
				inputs = append(inputs, raw)
			} else {
				notFound++
			}
			continue
		}
		if raw, ok := byEventID[ref.EventID]; ok {
			inputs = append(inputs, raw)
		} else {
			notFound++
		}
	}
	return inputs, notFound, nil
}
