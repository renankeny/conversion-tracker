package repo

import (
	"context"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type TrackableLinkStatsFilter struct {
	AccountID int64
	PixelID   int64
	PixelName string
	From      *time.Time
	To        *time.Time
}

// StatsByFilter lista os links rastreáveis (por Pixel ou da conta) enriquecidos com
// Cliques Humanos, Conversas Iniciadas e Vendas Confirmadas (deduplicadas por event_id
// e incluindo vendas em checkouts costuradas via group_id / whatsapp_chats).
func (r *TrackableLinkRepo) StatsByFilter(ctx context.Context, f TrackableLinkStatsFilter, fallbackAPIURL string) (*domain.TrackableLinksOverview, error) {
	conds := []string{"1=1"}
	args := make([]any, 0, 4)
	add := func(v any) string {
		args = append(args, v)
		return "$" + itoa(len(args))
	}

	if f.AccountID > 0 {
		conds = append(conds, "l.account_id = "+add(f.AccountID))
	}
	if f.PixelID > 0 {
		conds = append(conds, "l.pixel_id = "+add(f.PixelID))
	}
	if px := strings.TrimSpace(f.PixelName); px != "" {
		conds = append(conds, "p.name = "+add(px))
	}

	rows, err := r.pool.Query(ctx, `
		SELECT `+selectTrackableLinkCols+`
		FROM whatsapp_trackable_links l
		JOIN pixels p ON l.pixel_id = p.id
		LEFT JOIN whatsapp_instances wi ON l.whatsapp_instance_id = wi.id
		WHERE `+strings.Join(conds, " AND ")+`
		ORDER BY l.id DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]domain.TrackableLink, 0)
	byID := make(map[int64]int)
	linkIDs := make([]int64, 0)
	for rows.Next() {
		var item domain.TrackableLink
		if err := scanTrackableLink(rows, &item); err != nil {
			return nil, err
		}
		item.PublicURL = item.BuildPublicURL(fallbackAPIURL)
		byID[item.ID] = len(links)
		linkIDs = append(linkIDs, item.ID)
		links = append(links, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := &domain.TrackableLinksOverview{
		TotalLinks: len(links),
		Links:      links,
	}
	if len(linkIDs) == 0 {
		return out, nil
	}

	if err := r.populateClickCounts(ctx, linkIDs, f.From, f.To, links, byID); err != nil {
		return nil, err
	}
	if err := r.populateChatCounts(ctx, linkIDs, f.From, f.To, links, byID); err != nil {
		return nil, err
	}
	if err := r.populateSalesMetrics(ctx, f.AccountID, linkIDs, f.From, f.To, links, byID); err != nil {
		return nil, err
	}

	for _, l := range links {
		out.TotalClicks += l.ClicksCount
		out.TotalChats += l.ChatsCount
		out.TotalSales += l.SalesCount
		out.TotalRevenue += l.SalesRevenue
	}
	return out, nil
}

func (r *TrackableLinkRepo) populateClickCounts(
	ctx context.Context,
	linkIDs []int64,
	from, to *time.Time,
	links []domain.TrackableLink,
	byID map[int64]int,
) error {
	rows, err := r.pool.Query(ctx, `
		SELECT link_id, COUNT(*)
		FROM whatsapp_link_clicks
		WHERE link_id = ANY($1::bigint[])
		  AND is_bot = FALSE
		  AND ($2::timestamptz IS NULL OR created_at >= $2)
		  AND ($3::timestamptz IS NULL OR created_at < $3)
		GROUP BY link_id
	`, linkIDs, from, to)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var lid, cnt int64
		if err := rows.Scan(&lid, &cnt); err != nil {
			return err
		}
		if idx, ok := byID[lid]; ok {
			links[idx].ClicksCount = cnt
		}
	}
	return rows.Err()
}

func (r *TrackableLinkRepo) populateChatCounts(
	ctx context.Context,
	linkIDs []int64,
	from, to *time.Time,
	links []domain.TrackableLink,
	byID map[int64]int,
) error {
	rows, err := r.pool.Query(ctx, `
		SELECT trackable_link_id, COUNT(*)
		FROM whatsapp_chats
		WHERE trackable_link_id = ANY($1::bigint[])
		  AND ($2::timestamptz IS NULL OR created_at >= $2)
		  AND ($3::timestamptz IS NULL OR created_at < $3)
		GROUP BY trackable_link_id
	`, linkIDs, from, to)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var lid, cnt int64
		if err := rows.Scan(&lid, &cnt); err != nil {
			return err
		}
		if idx, ok := byID[lid]; ok {
			links[idx].ChatsCount = cnt
		}
	}
	return rows.Err()
}

func (r *TrackableLinkRepo) populateSalesMetrics(
	ctx context.Context,
	accountID int64,
	linkIDs []int64,
	from, to *time.Time,
	links []domain.TrackableLink,
	byID map[int64]int,
) error {
	rows, err := r.pool.Query(ctx, `
		WITH seed_users AS (
			SELECT id, account_id, external_id, group_id, trackable_link_id AS link_id
			FROM tracked_users
			WHERE trackable_link_id = ANY($1::bigint[])
			  AND ($2::bigint = 0 OR account_id = $2)
			UNION
			SELECT u.id, u.account_id, u.external_id, u.group_id, wc.trackable_link_id AS link_id
			FROM whatsapp_chats wc
			JOIN tracked_users u ON u.id = wc.tracked_user_id
			WHERE wc.trackable_link_id = ANY($1::bigint[])
			  AND ($2::bigint = 0 OR wc.account_id = $2)
		),
		all_linked_externals AS (
			SELECT DISTINCT s.account_id, s.external_id, s.link_id
			FROM seed_users s
			UNION
			SELECT DISTINCT ug.account_id, ug.external_id, s.link_id
			FROM seed_users s
			JOIN tracked_users ug ON ug.account_id = s.account_id AND ug.group_id = s.group_id
			WHERE s.group_id IS NOT NULL
		),
		dedup_sales AS (
			SELECT le.link_id, e.event_id, e.pixel_name, MAX(e.purchase_value) AS val
			FROM sent_events e
			JOIN all_linked_externals le
			  ON le.account_id = e.account_id AND le.external_id = e.external_id
			WHERE e.event_name = 'Purchase'
			  AND e.status = 'sent'
			  AND ($2::bigint = 0 OR e.account_id = $2)
			  AND ($3::timestamptz IS NULL OR e.created_at >= $3)
			  AND ($4::timestamptz IS NULL OR e.created_at < $4)
			GROUP BY le.link_id, e.event_id, e.pixel_name
		)
		SELECT link_id, COUNT(*), COALESCE(SUM(val), 0)
		FROM dedup_sales
		GROUP BY link_id
	`, linkIDs, accountID, from, to)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var lid, cnt int64
		var rev float64
		if err := rows.Scan(&lid, &cnt, &rev); err != nil {
			return err
		}
		if idx, ok := byID[lid]; ok {
			links[idx].SalesCount = cnt
			links[idx].SalesRevenue = rev
		}
	}
	return rows.Err()
}
