package campaigns

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/googleads"
	"github.com/renankeny/conversion-tracker/backend/internal/metaoauth"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

const campaignsCacheTTL = 3 * time.Minute

type OverviewParams struct {
	AccountID          int64
	PixelName          string
	Platform           string // "all" | "meta" | "google"
	AttributionMode    string // "last_click" | "first_click"
	Since              string // YYYY-MM-DD
	Until              string // YYYY-MM-DD
	From               time.Time
	To                 time.Time
	MetaAccountID      int64
	MetaAdAccountID    string
	GoogleAdsAccountID int64
	GoogleCustomerID   string
	ForceRefresh       bool
}

type Service struct {
	pixels    *repo.PixelRepo
	metaAccs  *repo.MetaAccountRepo
	gadsAccs  *repo.GoogleAdsAccountRepo
	metaOAuth *metaoauth.Client
	googleAds *googleads.Client
	attrRepo  *repo.CampaignsAttributionRepo
	rdb       *redis.Client
}

func NewService(
	pixels *repo.PixelRepo,
	metaAccs *repo.MetaAccountRepo,
	gadsAccs *repo.GoogleAdsAccountRepo,
	metaOAuth *metaoauth.Client,
	googleAds *googleads.Client,
	attrRepo *repo.CampaignsAttributionRepo,
	rdb *redis.Client,
) *Service {
	return &Service{
		pixels: pixels, metaAccs: metaAccs, gadsAccs: gadsAccs,
		metaOAuth: metaOAuth, googleAds: googleAds, attrRepo: attrRepo, rdb: rdb,
	}
}

func (s *Service) cacheKey(p OverviewParams) string {
	return fmt.Sprintf("tf:campaigns:v1:%d:%s:%s:%d:%s:%d:%s:%s:%s:%s",
		p.AccountID, p.Platform, p.PixelName, p.MetaAccountID, p.MetaAdAccountID,
		p.GoogleAdsAccountID, p.GoogleCustomerID, p.Since, p.Until, p.AttributionMode,
	)
}

func (s *Service) InvalidateAccountCache(ctx context.Context, accountID int64) {
	if s.rdb == nil {
		return
	}
	pattern := fmt.Sprintf("tf:campaigns:v1:%d:*", accountID)
	var cursor uint64
	for {
		keys, next, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			_ = s.rdb.Del(ctx, keys...).Err()
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
}

func (s *Service) Overview(ctx context.Context, p OverviewParams) (*domain.CampaignsOverviewResponse, error) {
	key := s.cacheKey(p)
	if !p.ForceRefresh && s.rdb != nil {
		if raw, err := s.rdb.Get(ctx, key).Bytes(); err == nil && len(raw) > 0 {
			var cached domain.CampaignsOverviewResponse
			if json.Unmarshal(raw, &cached) == nil {
				cached.Cached = true
				return &cached, nil
			}
		}
	}

	fin, metaTarget, gadsTarget := s.resolveTargets(ctx, p)
	var (
		metaBundle metaoauth.MetaHierarchyBundle
		gadsBundle googleads.HierarchyBundle
		events     []repo.AttributionEventRow
		warnings   []string
		warnMu     sync.Mutex
		wg         sync.WaitGroup
	)
	addWarn := func(msg string) {
		if strings.TrimSpace(msg) == "" {
			return
		}
		warnMu.Lock()
		warnings = append(warnings, msg)
		warnMu.Unlock()
	}

	if (p.Platform == "all" || p.Platform == "meta") && metaTarget.adAccountID != "" && s.metaOAuth != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := s.metaOAuth.FetchHierarchyInsights(ctx, metaTarget.acc, metaTarget.adAccountID, metaTarget.permToken, p.Since, p.Until)
			if err != nil {
				addWarn("Meta Ads v26.0: " + err.Error())
				return
			}
			metaBundle = b
		}()
	}

	if (p.Platform == "all" || p.Platform == "google") && gadsTarget.acc != nil && s.googleAds != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := s.googleAds.FetchHierarchyInsights(ctx, gadsTarget.acc, gadsTarget.customerID, gadsTarget.loginCID, p.Since, p.Until)
			if err != nil {
				addWarn(formatGoogleAdsCampaignWarn(err))
				return
			}
			gadsBundle = b
		}()
	}

	if s.attrRepo != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ev, err := s.attrRepo.FetchPeriodEvents(ctx, p.AccountID, p.PixelName, p.From, p.To)
			if err != nil {
				addWarn("Atribuição local: " + err.Error())
				return
			}
			events = ev
		}()
	}
	wg.Wait()

	camps := make([]domain.CampaignRow, 0, len(metaBundle.Campaigns)+len(gadsBundle.Campaigns)+1)
	adsets := make([]domain.CampaignRow, 0, len(metaBundle.AdSets)+len(gadsBundle.AdSets))
	ads := make([]domain.CampaignRow, 0, len(metaBundle.Ads)+len(gadsBundle.Ads))
	camps = append(camps, metaBundle.Campaigns...)
	camps = append(camps, gadsBundle.Campaigns...)
	adsets = append(adsets, metaBundle.AdSets...)
	adsets = append(adsets, gadsBundle.AdSets...)
	ads = append(ads, metaBundle.Ads...)
	ads = append(ads, gadsBundle.Ads...)

	unattr, hasUnattr := attributeEvents(camps, adsets, ads, events, p.AttributionMode)
	for i := range camps {
		computeRowFinancials(&camps[i], fin)
	}
	for i := range adsets {
		computeRowFinancials(&adsets[i], fin)
	}
	for i := range ads {
		computeRowFinancials(&ads[i], fin)
	}
	sortRowsBySpendAndRev(camps)
	sortRowsBySpendAndRev(adsets)
	sortRowsBySpendAndRev(ads)

	if hasUnattr {
		computeRowFinancials(&unattr, fin)
		camps = append(camps, unattr)
	}

	currency := metaBundle.Currency
	if currency == "" {
		currency = gadsBundle.Currency
	}
	if currency == "" {
		currency = "BRL"
	}

	resp := &domain.CampaignsOverviewResponse{
		Platform:        p.Platform,
		AttributionMode: p.AttributionMode,
		Since:           p.Since,
		Until:           p.Until,
		Currency:        currency,
		PixelFinancials: fin,
		Summary:         buildKpiSummary(camps, fin),
		Campaigns:       camps,
		AdSets:          adsets,
		Ads:             ads,
		Warnings:        warnings,
		Cached:          false,
	}
	if s.rdb != nil && len(warnings) == 0 {
		if raw, err := json.Marshal(resp); err == nil {
			_ = s.rdb.Set(ctx, key, raw, campaignsCacheTTL).Err()
		}
	}
	return resp, nil
}

func sortRowsBySpendAndRev(rows []domain.CampaignRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Spend != rows[j].Spend {
			return rows[i].Spend > rows[j].Spend
		}
		if rows[i].NetRevenue != rows[j].NetRevenue {
			return rows[i].NetRevenue > rows[j].NetRevenue
		}
		if (rows[i].Status == "ACTIVE") != (rows[j].Status == "ACTIVE") {
			return rows[i].Status == "ACTIVE"
		}
		return rows[i].Name < rows[j].Name
	})
}
