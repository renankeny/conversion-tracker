package handlers

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
)

const clearTokenSentinel = "__CLEAR__"

var pixelSlugPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func validatePixelInput(in pixelInput, requireAll bool) error {
	trimmedName := strings.TrimSpace(in.Name)
	if requireAll && trimmedName == "" {
		return errors.New("nome do pixel é obrigatório")
	}
	if trimmedName != "" && !pixelSlugPattern.MatchString(trimmedName) {
		return errors.New("nome do pixel deve usar apenas letras, números, hífen ou underline (máx. 64 caracteres)")
	}
	if requireAll {
		metaIDs := cleanStringSlice(in.PixelIDs)
		ttIDs := cleanStringSlice(in.TikTokPixelIDs)
		hasMeta := len(metaIDs) > 0
		hasMetaOAuth := in.MetaAccountID != nil && *in.MetaAccountID > 0
		hasTikTok := len(ttIDs) > 0
		hasGoogleAds := in.GoogleAdsAccountID != nil && *in.GoogleAdsAccountID > 0
		hasGA4 := in.GA4MeasurementID != nil && strings.TrimSpace(*in.GA4MeasurementID) != ""
		if hasMetaOAuth && !hasMeta && !hasTikTok && !hasGoogleAds && !hasGA4 {
			return errors.New("selecione pelo menos um Pixel ID da Meta para disparar eventos via OAuth")
		}
		if !hasMeta && !hasMetaOAuth && !hasTikTok && !hasGoogleAds && !hasGA4 {
			return errors.New("configure pelo menos um canal (Meta, TikTok, Google Ads ou GA4)")
		}
		if hasMeta && !hasMetaOAuth && len(in.AccessTokens) != len(metaIDs) {
			return errors.New("informe o Access Token para cada Pixel ID da Meta")
		}
		if hasTikTok && len(in.TikTokAccessTokens) != len(ttIDs) {
			return errors.New("informe o Access Token para cada Pixel ID do TikTok")
		}
	}
	return nil
}

func applyOptionalPixelFields(p *domain.Pixel, in pixelInput) {
	if in.Enabled != nil {
		p.Enabled = *in.Enabled
	}
	if in.TrackingHost != nil {
		p.TrackingHost = normalizeHost(*in.TrackingHost)
	}
	if in.GA4MeasurementID != nil {
		p.GA4MeasurementID = strings.TrimSpace(*in.GA4MeasurementID)
	}
	if in.GA4APISecret != nil {
		val := strings.TrimSpace(*in.GA4APISecret)
		if val == clearTokenSentinel || (val == "" && p.GA4MeasurementID == "") {
			p.GA4APISecret = ""
		} else if !isMaskedOrEmpty(val) {
			p.GA4APISecret = val
		}
	}
	if in.MetaPermanentToken != nil {
		val := strings.TrimSpace(*in.MetaPermanentToken)
		if val == clearTokenSentinel {
			p.MetaPermanentToken = ""
		} else if !isMaskedOrEmpty(val) {
			p.MetaPermanentToken = val
		}
	}
	if in.MetaAdAccountID != nil {
		p.MetaAdAccountID = strings.TrimSpace(*in.MetaAdAccountID)
	}
	if in.GoogleAdsCustomerID != nil {
		p.GoogleAdsCustomerID = strings.TrimSpace(*in.GoogleAdsCustomerID)
		if p.GoogleAdsCustomerID == "" {
			p.GoogleAdsCustomerName = ""
			p.GoogleAdsLoginCustomerID = ""
		}
	}
	if in.GoogleAdsCustomerName != nil {
		p.GoogleAdsCustomerName = strings.TrimSpace(*in.GoogleAdsCustomerName)
	}
	if in.GoogleAdsLoginCustomerID != nil {
		p.GoogleAdsLoginCustomerID = strings.TrimSpace(*in.GoogleAdsLoginCustomerID)
	}
	if p.GoogleAdsAccountID == nil {
		p.GoogleAdsCustomerID = ""
		p.GoogleAdsCustomerName = ""
		p.GoogleAdsLoginCustomerID = ""
	}
	if in.AggregationEnabled != nil {
		p.AggregationEnabled = *in.AggregationEnabled
	}
	if in.AggregationWindowMinutes != nil {
		p.AggregationWindowMinutes = *in.AggregationWindowMinutes
	}
	if in.SalesGoalEnabled != nil {
		p.SalesGoalEnabled = *in.SalesGoalEnabled
	}
	if in.SalesGoalType != nil {
		p.SalesGoalType = domain.NormalizeSalesGoalType(strings.TrimSpace(*in.SalesGoalType))
	}
	if in.SalesGoalTarget != nil {
		val := *in.SalesGoalTarget
		if val < 0 {
			val = 0
		}
		p.SalesGoalTarget = val
	}
	if in.SalesGoalTarget != nil || in.SalesGoalMonths != nil {
		domain.SyncPixelSalesGoalMonths(p, in.SalesGoalMonths)
	}
	if in.PublicShowSalesGoal != nil {
		v := *in.PublicShowSalesGoal
		p.DashboardConfig.PublicShowSalesGoal = &v
	}
}

func syncPixelGoogleAdsDefaults(ctx context.Context, gads *repo.GoogleAdsAccountRepo, p *domain.Pixel) {
	if gads == nil || p == nil || p.GoogleAdsAccountID == nil || *p.GoogleAdsAccountID <= 0 {
		return
	}
	if strings.TrimSpace(p.GoogleAdsCustomerID) != "" {
		return
	}
	acc, err := gads.FindByID(ctx, *p.GoogleAdsAccountID)
	if err != nil || acc == nil || strings.TrimSpace(acc.CustomerID) == "" {
		return
	}
	p.GoogleAdsCustomerID = strings.TrimSpace(acc.CustomerID)
	if name := strings.TrimSpace(acc.CustomerName); name != "" {
		p.GoogleAdsCustomerName = name
	} else {
		p.GoogleAdsCustomerName = strings.TrimSpace(acc.Name)
	}
	p.GoogleAdsLoginCustomerID = strings.TrimSpace(acc.LoginCustomerID)
}

func syncPixelMetaAdsDefaults(ctx context.Context, metaAccs *repo.MetaAccountRepo, p *domain.Pixel) {
	if metaAccs == nil || p == nil || p.MetaAccountID == nil || *p.MetaAccountID <= 0 {
		return
	}
	if strings.TrimSpace(p.MetaAdAccountID) != "" {
		return
	}
	acc, err := metaAccs.FindByID(ctx, *p.MetaAccountID)
	if err != nil || acc == nil || strings.TrimSpace(acc.AdAccountID) == "" {
		return
	}
	p.MetaAdAccountID = strings.TrimSpace(acc.AdAccountID)
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func cleanStringSlice(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if v := strings.TrimSpace(s); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func nilIfZeroInt64(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

func cleanDomains(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, d := range in {
		d = strings.ToLower(strings.TrimSpace(d))
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

func normalizeHost(in string) string {
	h := strings.TrimSpace(in)
	if h == "" {
		return ""
	}
	h = strings.ToLower(h)
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	if i := strings.IndexAny(h, "/?#"); i != -1 {
		h = h[:i]
	}
	return strings.TrimSuffix(h, ".")
}
