package repo

import "context"

// populateLTV preenche GroupedEvent.LTV com o valor total histórico de compras
// da pessoa dona de cada external_id presente em `out`, filtrando estritamente
// por accountID e apenas pelos external_ids relevantes da página.
func (r *SentEventRepo) populateLTV(ctx context.Context, accountID int64, out []GroupedEvent) error {
	extIDs := uniqueGroupedExtIDs(out)
	if len(extIDs) == 0 {
		return nil
	}

	gkeyByExtID, err := r.groupKeysFor(ctx, accountID, extIDs)
	if err != nil {
		return err
	}

	rows, err := r.pool.Query(ctx, `
		WITH keys AS (
			SELECT DISTINCT account_id, COALESCE(group_id, id) AS gkey
			FROM tracked_users
			WHERE external_id = ANY($1::text[])
			  AND ($2::bigint = 0 OR account_id = $2)
		),
		members AS (
			SELECT t.account_id, t.external_id, COALESCE(t.group_id, t.id) AS gkey
			FROM tracked_users t
			JOIN keys k ON k.gkey = COALESCE(t.group_id, t.id) AND k.account_id = t.account_id
		)
		SELECT m.gkey, COALESCE(SUM(s.max_value), 0)
		FROM members m
		LEFT JOIN (
			SELECT se.account_id, se.external_id, se.event_id, MAX(se.purchase_value) AS max_value
			FROM sent_events se
			WHERE se.event_name = 'Purchase'
			  AND se.status = 'sent'
			  AND ($2::bigint = 0 OR se.account_id = $2)
			  AND se.external_id IN (SELECT external_id FROM members)
			GROUP BY se.account_id, se.external_id, se.event_id
		) s ON s.external_id = m.external_id AND s.account_id = m.account_id
		GROUP BY m.gkey
	`, extIDs, accountID)
	if err != nil {
		return err
	}
	defer rows.Close()

	ltvByGKey := make(map[int64]float64, len(extIDs))
	for rows.Next() {
		var gkey int64
		var ltv float64
		if err := rows.Scan(&gkey, &ltv); err != nil {
			return err
		}
		ltvByGKey[gkey] = ltv
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range out {
		if gkey, ok := gkeyByExtID[out[i].ExternalID]; ok {
			if v, ok := ltvByGKey[gkey]; ok {
				ltv := v
				out[i].LTV = &ltv
			}
		}
	}
	return nil
}

func (r *SentEventRepo) groupKeysFor(ctx context.Context, accountID int64, extIDs []string) (map[string]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT external_id, COALESCE(group_id, id)
		FROM tracked_users
		WHERE external_id = ANY($1::text[])
		  AND ($2::bigint = 0 OR account_id = $2)
	`, extIDs, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64, len(extIDs))
	for rows.Next() {
		var extID string
		var gkey int64
		if err := rows.Scan(&extID, &gkey); err != nil {
			return nil, err
		}
		out[extID] = gkey
	}
	return out, rows.Err()
}

func (r *SentEventRepo) populateUserContext(ctx context.Context, accountID int64, out []GroupedEvent) error {
	extIDs := uniqueGroupedExtIDs(out)
	if len(extIDs) == 0 {
		return nil
	}

	gkeyByExtID, err := r.groupKeysFor(ctx, accountID, extIDs)
	if err != nil {
		return err
	}

	rows, err := r.pool.Query(ctx, `
		WITH keys AS (
			SELECT DISTINCT account_id, COALESCE(group_id, id) AS gkey
			FROM tracked_users
			WHERE external_id = ANY($1::text[])
			  AND ($2::bigint = 0 OR account_id = $2)
		),
		members AS (
			SELECT t.*, COALESCE(t.group_id, t.id) AS gkey
			FROM tracked_users t
			JOIN keys k ON k.gkey = COALESCE(t.group_id, t.id) AND k.account_id = t.account_id
		)
		SELECT k.gkey,
			COALESCE(
				NULLIF((SELECT first_utm_source FROM members m WHERE m.gkey = k.gkey AND m.first_utm_source <> '' ORDER BY m.created_at ASC LIMIT 1), ''),
				NULLIF((SELECT last_utm_source FROM members m WHERE m.gkey = k.gkey AND m.last_utm_source <> '' ORDER BY m.created_at ASC LIMIT 1), ''),
				(SELECT m.first_utm_source FROM members m WHERE m.gkey = k.gkey ORDER BY m.created_at ASC LIMIT 1),
				''
			) AS group_first_source,
			COALESCE(
				NULLIF((SELECT last_utm_source FROM members m WHERE m.gkey = k.gkey AND m.last_utm_source <> '' ORDER BY m.updated_at DESC LIMIT 1), ''),
				NULLIF((SELECT first_utm_source FROM members m WHERE m.gkey = k.gkey AND m.first_utm_source <> '' ORDER BY m.updated_at DESC LIMIT 1), ''),
				(SELECT m.last_utm_source FROM members m WHERE m.gkey = k.gkey ORDER BY m.updated_at DESC LIMIT 1),
				''
			) AS group_last_source
		FROM keys k
	`, extIDs, accountID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type ctxInfo struct{ first, last string }
	byGKey := make(map[int64]ctxInfo, len(extIDs))
	for rows.Next() {
		var gkey int64
		var info ctxInfo
		if err := rows.Scan(&gkey, &info.first, &info.last); err != nil {
			return err
		}
		byGKey[gkey] = info
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range out {
		if gkey, ok := gkeyByExtID[out[i].ExternalID]; ok {
			if info, ok := byGKey[gkey]; ok {
				out[i].FirstUTMSource = info.first
				out[i].LastUTMSource = info.last
			}
		}
	}
	return nil
}

func uniqueGroupedExtIDs(out []GroupedEvent) []string {
	extIDs := make([]string, 0, len(out))
	seen := make(map[string]bool, len(out))
	for _, ev := range out {
		if ev.ExternalID != "" && !seen[ev.ExternalID] {
			seen[ev.ExternalID] = true
			extIDs = append(extIDs, ev.ExternalID)
		}
	}
	return extIDs
}
