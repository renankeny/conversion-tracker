package queue

import (
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func ClassifyPortalOrigin(ctwaClid, sourceID string, u *domain.TrackedUser) string {
	if ctwaClid != "" || sourceID != "" {
		return domain.WhatsAppOriginMetaCTWA
	}
	if u == nil {
		return domain.WhatsAppOriginWebsiteOrganic
	}

	isFromTrackableLink := u.TrackableLinkID != nil && *u.TrackableLinkID > 0

	// Prioriza o último toque pago quando a pessoa interagiu com mais de uma rede ao longo do tempo
	if isGoogleSource(u.LastUTMSource) && (u.GCLID != "" || u.GBraid != "" || u.WBraid != "" || isPaidMedium(u.LastUTMMedium) || (!isFromTrackableLink && !isOrganicMedium(u.LastUTMMedium) && (isAdsOnlyGoogleSource(u.LastUTMSource) || u.LastUTMCampaign != ""))) {
		return domain.WhatsAppOriginGoogleAds
	}
	if isTikTokSource(u.LastUTMSource) && (u.TTClid != "" || isPaidMedium(u.LastUTMMedium) || (!isFromTrackableLink && !isOrganicMedium(u.LastUTMMedium) && u.LastUTMCampaign != "")) {
		return domain.WhatsAppOriginTikTokAds
	}
	if isMetaSource(u.LastUTMSource) && (u.FBC != "" || isPaidMedium(u.LastUTMMedium) || (!isFromTrackableLink && !isOrganicMedium(u.LastUTMMedium) && (isAdsOnlyMetaSource(u.LastUTMSource) || u.LastUTMCampaign != ""))) {
		return domain.WhatsAppOriginMetaLP
	}

	if u.CtwaClid != "" || u.WaAdID != "" {
		return domain.WhatsAppOriginMetaCTWA
	}
	if u.GCLID != "" || u.GBraid != "" || u.WBraid != "" || isPaidGoogleUTM(u.FirstUTMSource, u.FirstUTMMedium) || isPaidGoogleUTM(u.LastUTMSource, u.LastUTMMedium) {
		return domain.WhatsAppOriginGoogleAds
	}
	if u.TTClid != "" || isPaidTikTokUTM(u.FirstUTMSource, u.FirstUTMMedium) || isPaidTikTokUTM(u.LastUTMSource, u.LastUTMMedium) {
		return domain.WhatsAppOriginTikTokAds
	}
	if u.FBC != "" || isPaidMetaUTM(u.FirstUTMSource, u.FirstUTMMedium) || isPaidMetaUTM(u.LastUTMSource, u.LastUTMMedium) {
		return domain.WhatsAppOriginMetaLP
	}
	if isFromTrackableLink || isBioLinkSession(u) {
		return domain.WhatsAppOriginBioLinkOrganic
	}
	return domain.WhatsAppOriginWebsiteOrganic
}

func isPaidMedium(med string) bool {
	m := strings.ToLower(strings.TrimSpace(med))
	if m == "" {
		return false
	}
	switch m {
	case "cpc", "ppc", "paid", "ads", "ad", "cpm", "cpa", "paid_social", "paidsocial",
		"paid-social", "social_paid", "pmax", "display", "remarketing", "retargeting",
		"ctwa", "sponsored", "impulsionar", "impulsionamento", "trafego_pago", "trafego-pago":
		return true
	}
	return strings.Contains(m, "pago") || strings.HasPrefix(m, "paid_") || strings.HasSuffix(m, "_ads")
}

func isOrganicMedium(med string) bool {
	switch strings.ToLower(strings.TrimSpace(med)) {
	case "organic", "organico", "orgânico", "social", "social-network", "social_organic",
		"referral", "referencia", "bio", "link_rastreavel", "linktree", "perfil",
		"qr", "qrcode", "direct", "direto", "none", "(none)", "email", "newsletter", "seo":
		return true
	default:
		return false
	}
}

func isAdsOnlyMetaSource(src string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	return s == "meta_ads" || s == "facebook_ads" || s == "fb_ads" || s == "ig_ads" || s == "instagram_ads" || s == "an"
}

func isAdsOnlyGoogleSource(src string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	return s == "adwords" || s == "gads" || s == "google_ads" || s == "googleads"
}

func isMetaSource(src string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	return s == "facebook" || s == "fb" || s == "instagram" || s == "ig" || s == "meta" || isAdsOnlyMetaSource(s) || strings.Contains(s, "facebook") || strings.Contains(s, "instagram")
}

func isGoogleSource(src string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	return s == "google" || s == "youtube" || isAdsOnlyGoogleSource(s) || strings.Contains(s, "google") || strings.Contains(s, "adwords")
}

func isTikTokSource(src string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	return s == "tiktok" || s == "tt" || s == "tiktok_ads" || strings.Contains(s, "tiktok") || strings.Contains(s, "bytedance")
}

func isPaidMetaUTM(src, med string) bool {
	return isAdsOnlyMetaSource(src) || (isMetaSource(src) && isPaidMedium(med))
}

func isPaidGoogleUTM(src, med string) bool {
	return isAdsOnlyGoogleSource(src) || (isGoogleSource(src) && isPaidMedium(med))
}

func isPaidTikTokUTM(src, med string) bool {
	return strings.EqualFold(strings.TrimSpace(src), "tiktok_ads") || (isTikTokSource(src) && isPaidMedium(med))
}

func isBioLinkSession(u *domain.TrackedUser) bool {
	if u == nil {
		return false
	}
	med := strings.ToLower(u.FirstUTMMedium + " " + u.LastUTMMedium + " " + u.FirstUTMSource + " " + u.LastUTMSource)
	if strings.Contains(med, "bio") || strings.Contains(med, "linktree") || strings.Contains(med, "perfil") || strings.Contains(med, "link_rastreavel") {
		return true
	}
	urls := strings.ToLower(u.FirstURL + " " + u.URL + " " + u.Referrer)
	for _, token := range []string{"linktr.ee", "beacons.ai", "bio.site", "lnk.bio", "l.instagram.com", "instagram.com", "/bio"} {
		if strings.Contains(urls, token) {
			return true
		}
	}
	return false
}

