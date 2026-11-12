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

	// Prioriza o último toque pago quando a pessoa interagiu com mais de uma rede ao longo do tempo
	if isGoogleSource(u.LastUTMSource) && (u.GCLID != "" || u.GBraid != "" || u.WBraid != "" || isPaidMedium(u.LastUTMMedium) || u.LastUTMCampaign != "") {
		return domain.WhatsAppOriginGoogleAds
	}
	if isTikTokSource(u.LastUTMSource) && (u.TTClid != "" || isPaidMedium(u.LastUTMMedium) || u.LastUTMCampaign != "") {
		return domain.WhatsAppOriginTikTokAds
	}
	if isMetaSource(u.LastUTMSource) && (u.FBC != "" || isPaidMedium(u.LastUTMMedium) || u.LastUTMCampaign != "") {
		return domain.WhatsAppOriginMetaLP
	}

	if u.CtwaClid != "" {
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
	if isBioLinkSession(u) {
		return domain.WhatsAppOriginBioLinkOrganic
	}
	return domain.WhatsAppOriginWebsiteOrganic
}

func isPaidMedium(med string) bool {
	m := strings.ToLower(strings.TrimSpace(med))
	if m == "" {
		return false
	}
	return m == "cpc" || m == "ppc" || m == "paid" || m == "ads" || m == "cpm" || m == "paid_social" || m == "pmax" || strings.Contains(m, "ad") || strings.Contains(m, "pago") || strings.Contains(m, "trafego")
}

func isMetaSource(src string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	return s == "facebook" || s == "fb" || s == "instagram" || s == "ig" || s == "meta" || strings.Contains(s, "facebook") || strings.Contains(s, "instagram")
}

func isGoogleSource(src string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	return s == "google" || s == "adwords" || s == "youtube" || s == "gads" || strings.Contains(s, "google") || strings.Contains(s, "adwords")
}

func isTikTokSource(src string) bool {
	s := strings.ToLower(strings.TrimSpace(src))
	return s == "tiktok" || s == "tt" || strings.Contains(s, "tiktok") || strings.Contains(s, "bytedance")
}

func isPaidMetaUTM(src, med string) bool {
	return isMetaSource(src) && isPaidMedium(med)
}

func isPaidGoogleUTM(src, med string) bool {
	return isGoogleSource(src) && isPaidMedium(med)
}

func isPaidTikTokUTM(src, med string) bool {
	return isTikTokSource(src) && isPaidMedium(med)
}

func isBioLinkSession(u *domain.TrackedUser) bool {
	if u == nil {
		return false
	}
	med := strings.ToLower(u.FirstUTMMedium + " " + u.LastUTMMedium + " " + u.FirstUTMSource + " " + u.LastUTMSource)
	if strings.Contains(med, "bio") || strings.Contains(med, "linktree") || strings.Contains(med, "perfil") {
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

