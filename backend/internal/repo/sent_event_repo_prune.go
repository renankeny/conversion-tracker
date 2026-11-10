package repo

import (
	"context"
	"fmt"
	"time"
)

// FindByEventID retorna todas as linhas de sent_events com aquele
// event_id + pixel_name (1 por canal). Usado pelo modal de detalhes
// pra mostrar payload + response de cada canal.
func (r *SentEventRepo) FindByEventID(ctx context.Context, eventID, pixelName string, accountID int64) ([]SentEvent, error) {
	args := []any{eventID}
	where := "WHERE event_id = $1"
	if pixelName != "" {
		args = append(args, pixelName)
		where += fmt.Sprintf(" AND pixel_name = $%d", len(args))
	}
	if accountID != 0 {
		args = append(args, accountID)
		where += fmt.Sprintf(" AND (account_id = $%d OR account_id = 0 OR account_id IS NULL)", len(args))
	}
	q := "SELECT DISTINCT ON (channel) id, event_id, event_name, pixel_name, external_id, channel, status, " +
		"response_code, response_body, error, COALESCE(payload::text, '') AS payload, " +
		"purchase_value, created_at, updated_at " +
		"FROM sent_events " + where + " ORDER BY channel, id DESC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]SentEvent, 0, 3)
	for rows.Next() {
		var e SentEvent
		if err := rows.Scan(&e.ID, &e.EventID, &e.EventName, &e.PixelName,
			&e.ExternalID, &e.Channel, &e.Status, &e.ResponseCode, &e.ResponseBody, &e.Error,
			&e.Payload, &e.PurchaseValue, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteByEventIDs remove todas as linhas de sent_events com os event_ids
// informados. Aplica scope multi-tenant: se accountID != 0, só apaga linhas
// da conta. Retorna o número de linhas removidas.
func (r *SentEventRepo) DeleteByEventIDs(ctx context.Context, accountID int64, eventIDs []string) (int64, error) {
	if len(eventIDs) == 0 {
		return 0, nil
	}
	args := []any{eventIDs}
	where := "WHERE event_id = ANY($1::text[])"
	if accountID != 0 {
		args = append(args, accountID)
		where += " AND account_id = $2"
	}
	tag, err := r.pool.Exec(ctx, "DELETE FROM sent_events "+where, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// FindInputsByEventIDs carrega o input_json de um evento por event_id.
// Retorna apenas a primeira linha com input_json não nulo por event_id
// (todos os canais têm o mesmo input).
func (r *SentEventRepo) FindInputsByEventIDs(ctx context.Context, accountID int64, eventIDs []string) (map[string][]byte, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}
	args := []any{eventIDs}
	where := "WHERE event_id = ANY($1::text[]) AND input_json IS NOT NULL"
	if accountID != 0 {
		args = append(args, accountID)
		where += " AND account_id = $2"
	}
	q := "SELECT DISTINCT ON (event_id) event_id, input_json::text FROM sent_events " +
		where + " ORDER BY event_id, id DESC"

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]byte, len(eventIDs))
	for rows.Next() {
		var eventID string
		var raw []byte
		if err := rows.Scan(&eventID, &raw); err != nil {
			return nil, err
		}
		result[eventID] = raw
	}
	return result, rows.Err()
}

// itoa minimal pra montar placeholders sem importar strconv só pra isso
// (evita uma alocação por chamada).
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	const digits = "0123456789"
	buf := make([]byte, 0, 4)
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		buf = append([]byte{digits[i%10]}, buf...)
		i /= 10
	}
	if neg {
		return "-" + string(buf)
	}
	return string(buf)
}

// SalesConfirmationItem holds data for manual sales confirmation UI
type SalesConfirmationItem struct {
	EventID       string    `json:"event_id"`
	EventName     string    `json:"event_name"`
	PixelName     string    `json:"pixel_name"`
	ExternalID    string    `json:"external_id"`
	PurchaseValue float64   `json:"purchase_value"`
	CreatedAt     time.Time `json:"created_at"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
}

// ListSalesForConfirmation retrieves grouped AddPaymentInfo (Pix) and Purchase events
// along with customer data for manual confirmation.
func (r *SentEventRepo) ListSalesForConfirmation(ctx context.Context, accountID int64, pixelName string, limit, offset int) ([]SalesConfirmationItem, int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	countQ := `
		SELECT COUNT(DISTINCT event_id)
		FROM sent_events
		WHERE event_name IN ('AddPaymentInfo', 'Purchase')
		  AND (account_id = $1 OR $1 = 0 OR account_id IS NULL)
		  AND ($2 = '' OR pixel_name = $2)
	`
	var total int64
	if err := r.pool.QueryRow(ctx, countQ, accountID, pixelName).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := `
		SELECT * FROM (
			SELECT DISTINCT ON (e.event_id)
				e.event_id, e.event_name, e.pixel_name, e.external_id, COALESCE(e.purchase_value, 0), e.created_at,
				COALESCE(u.first_name, ''), COALESCE(u.last_name, ''), COALESCE(u.email, ''), COALESCE(u.phone, '')
			FROM sent_events e
			LEFT JOIN tracked_users u ON e.external_id = u.external_id AND e.pixel_name = u.pixel_name
			WHERE e.event_name IN ('AddPaymentInfo', 'Purchase')
			  AND (e.account_id = $1 OR $1 = 0 OR e.account_id IS NULL)
			  AND ($2 = '' OR e.pixel_name = $2)
			ORDER BY e.event_id, (CASE WHEN e.event_name = 'Purchase' THEN 1 ELSE 2 END) ASC, e.created_at DESC
		) t
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.pool.Query(ctx, q, accountID, pixelName, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []SalesConfirmationItem
	for rows.Next() {
		var item SalesConfirmationItem
		if err := rows.Scan(
			&item.EventID, &item.EventName, &item.PixelName, &item.ExternalID,
			&item.PurchaseValue, &item.CreatedAt,
			&item.FirstName, &item.LastName, &item.Email, &item.Phone,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

// MarkAsConfirmed atualiza os eventos com o eventID para 'Purchase' (confirmado manualmente pelo admin).
func (r *SentEventRepo) MarkAsConfirmed(ctx context.Context, eventID, pixelName string, accountID int64) error {
	if eventID == "" {
		return nil
	}
	var err error
	if accountID != 0 {
		_, err = r.pool.Exec(ctx, `
			UPDATE sent_events
			SET event_name = 'Purchase', updated_at = NOW()
			WHERE event_id = $1 AND ($2 = '' OR pixel_name = $2) AND (account_id = $3 OR account_id = 0 OR account_id IS NULL)
		`, eventID, pixelName, accountID)
	} else {
		_, err = r.pool.Exec(ctx, `
			UPDATE sent_events
			SET event_name = 'Purchase', updated_at = NOW()
			WHERE event_id = $1 AND ($2 = '' OR pixel_name = $2)
		`, eventID, pixelName)
	}
	return err
}
