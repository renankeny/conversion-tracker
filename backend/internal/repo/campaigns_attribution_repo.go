package repo

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AttributionEventRow representa um evento rastreado (venda, reembolso, lead ou conversa WA)
// enriquecido com chaves de atribuição de plataforma (IDs CTWA) e UTMs de 1º e último clique.
type AttributionEventRow struct {
	Kind       string  // "sale" | "refund" | "chargeback" | "lead" | "conversation"
	Value      float64
	AIQualified bool

	CampaignID string
	AdSetID    string
	AdID       string

	FirstUTMSource   string
	FirstUTMMedium   string
	FirstUTMCampaign string
	FirstUTMContent  string
	FirstUTMTerm     string

	LastUTMSource   string
	LastUTMMedium   string
	LastUTMCampaign string
	LastUTMContent  string
	LastUTMTerm     string

	HasGoogleClick bool
	HasMetaClick   bool
}

type CampaignsAttributionRepo struct {
	pool *pgxpool.Pool
}

func NewCampaignsAttributionRepo(pool *pgxpool.Pool) *CampaignsAttributionRepo {
	return &CampaignsAttributionRepo{pool: pool}
}

// FetchPeriodEvents carrega vendas aprovadas, reembolsos/chargebacks, leads e conversas WhatsApp
// do período em paralelo para cruzamento exato com Meta v26.0 e Google Ads v26.
func (r *CampaignsAttributionRepo) FetchPeriodEvents(
	ctx context.Context,
	accountID int64,
	pixelName string,
	from, to time.Time,
) ([]AttributionEventRow, error) {
	var (
		sales, refunds, leads, chats []AttributionEventRow
		wg                           sync.WaitGroup
		mu                           sync.Mutex
		firstErr                     error
	)
	recordErr := func(err error) {
		if err == nil {
			return
		}
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}

	wg.Add(4)
	go func() {
		defer wg.Done()
		rows, err := r.querySales(ctx, accountID, pixelName, from, to)
		recordErr(err)
		sales = rows
	}()
	go func() {
		defer wg.Done()
		rows, err := r.queryRefunds(ctx, accountID, pixelName, from, to)
		recordErr(err)
		refunds = rows
	}()
	go func() {
		defer wg.Done()
		rows, err := r.queryLeads(ctx, accountID, pixelName, from, to)
		recordErr(err)
		leads = rows
	}()
	go func() {
		defer wg.Done()
		rows, err := r.queryConversations(ctx, accountID, pixelName, from, to)
		recordErr(err)
		chats = rows
	}()
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	out := make([]AttributionEventRow, 0, len(sales)+len(refunds)+len(leads)+len(chats))
	out = append(out, sales...)
	out = append(out, refunds...)
	out = append(out, leads...)
	out = append(out, chats...)
	return out, nil
}

func (r *CampaignsAttributionRepo) querySales(ctx context.Context, accountID int64, pixelName string, from, to time.Time) ([]AttributionEventRow, error) {
	const q = `
		WITH dedup_sales AS (
			SELECT e.account_id, e.pixel_name, e.event_id,
			       MAX(e.purchase_value) AS val,
			       MAX(e.external_id) AS external_id
			FROM sent_events e
			WHERE e.event_name = 'Purchase'
			  AND e.status = 'sent'
			  AND ($1 = 0 OR e.account_id = $1)
			  AND ($2 = '' OR e.pixel_name = $2)
			  AND e.created_at >= $3 AND e.created_at < $4
			GROUP BY e.account_id, e.pixel_name, e.event_id
		)
		SELECT
			ds.val,
			COALESCE(wc.campaign_id, ''), COALESCE(wc.adset_id, ''), COALESCE(NULLIF(wc.source_id, ''), u.wa_ad_id, ''),
			COALESCE(u.first_utm_source, ''), COALESCE(u.first_utm_medium, ''), COALESCE(u.first_utm_campaign, ''),
			COALESCE(u.first_utm_content, ''), COALESCE(u.first_utm_term, ''),
			COALESCE(u.last_utm_source, ''), COALESCE(u.last_utm_medium, ''), COALESCE(u.last_utm_campaign, ''),
			COALESCE(u.last_utm_content, ''), COALESCE(u.last_utm_term, ''),
			(COALESCE(u.gclid, '') <> '' OR COALESCE(u.wbraid, '') <> '' OR COALESCE(u.gbraid, '') <> ''),
			(COALESCE(u.fbc, '') <> '' OR COALESCE(u.ctwa_clid, '') <> '' OR COALESCE(u.wa_ad_id, '') <> '' OR COALESCE(wc.source_id, '') <> '')
		FROM dedup_sales ds
		LEFT JOIN tracked_users u ON u.account_id = ds.account_id AND u.external_id = ds.external_id
		LEFT JOIN LATERAL (
			SELECT campaign_id, adset_id, source_id
			FROM whatsapp_chats
			WHERE account_id = ds.account_id
			  AND tracked_user_id = u.id
			  AND (campaign_id <> '' OR adset_id <> '' OR source_id <> '')
			ORDER BY updated_at DESC LIMIT 1
		) wc ON u.id IS NOT NULL
	`
	return r.scanAttrRows(ctx, "sale", q, accountID, pixelName, from, to)
}

func (r *CampaignsAttributionRepo) queryRefunds(ctx context.Context, accountID int64, pixelName string, from, to time.Time) ([]AttributionEventRow, error) {
	const q = `
		SELECT
			sr.refund_type, sr.amount,
			COALESCE(wc.campaign_id, ''), COALESCE(wc.adset_id, ''), COALESCE(NULLIF(wc.source_id, ''), u.wa_ad_id, ''),
			COALESCE(u.first_utm_source, ''), COALESCE(u.first_utm_medium, ''), COALESCE(u.first_utm_campaign, ''),
			COALESCE(u.first_utm_content, ''), COALESCE(u.first_utm_term, ''),
			COALESCE(u.last_utm_source, ''), COALESCE(u.last_utm_medium, ''), COALESCE(u.last_utm_campaign, ''),
			COALESCE(u.last_utm_content, ''), COALESCE(u.last_utm_term, ''),
			(COALESCE(u.gclid, '') <> '' OR COALESCE(u.wbraid, '') <> '' OR COALESCE(u.gbraid, '') <> ''),
			(COALESCE(u.fbc, '') <> '' OR COALESCE(u.ctwa_clid, '') <> '' OR COALESCE(u.wa_ad_id, '') <> '' OR COALESCE(wc.source_id, '') <> '')
		FROM sale_refunds sr
		LEFT JOIN tracked_users u ON u.id = sr.tracked_user_id
		LEFT JOIN LATERAL (
			SELECT campaign_id, adset_id, source_id
			FROM whatsapp_chats
			WHERE account_id = sr.account_id
			  AND tracked_user_id = u.id
			  AND (campaign_id <> '' OR adset_id <> '' OR source_id <> '')
			ORDER BY updated_at DESC LIMIT 1
		) wc ON u.id IS NOT NULL
		WHERE ($1 = 0 OR sr.account_id = $1)
		  AND ($2 = '' OR sr.pixel_name = $2)
		  AND sr.refunded_at >= $3 AND sr.refunded_at < $4
	`
	rows, err := r.pool.Query(ctx, q, accountID, pixelName, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AttributionEventRow
	for rows.Next() {
		var row AttributionEventRow
		var rType string
		if err := rows.Scan(
			&rType, &row.Value,
			&row.CampaignID, &row.AdSetID, &row.AdID,
			&row.FirstUTMSource, &row.FirstUTMMedium, &row.FirstUTMCampaign, &row.FirstUTMContent, &row.FirstUTMTerm,
			&row.LastUTMSource, &row.LastUTMMedium, &row.LastUTMCampaign, &row.LastUTMContent, &row.LastUTMTerm,
			&row.HasGoogleClick, &row.HasMetaClick,
		); err != nil {
			return nil, err
		}
		if strings.EqualFold(rType, "chargeback") {
			row.Kind = "chargeback"
		} else {
			row.Kind = "refund"
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
