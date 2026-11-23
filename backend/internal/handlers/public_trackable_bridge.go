package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type bridgeRenderParams struct {
	Title              string
	Phone              string
	PrefillMessage     string
	ProtocolCode       string
	MetaBrowserPixelID string
	ExternalID         string
}

var previewBotTokens = []string{
	"facebookexternalhit", "facebot", "whatsapp", "telegrambot",
	"twitterbot", "discordbot", "slackbot", "linkedinbot",
	"pinterest", "skypeuripreview", "googlebot", "bingbot", "applebot",
}

// IsLinkPreviewCrawler detecta crawlers de pré-visualização de links (Instagram Direct,
// Bio pre-fetch, WhatsApp preview, Telegram, etc.) sem bloquear humanos em In-App Browsers.
func IsLinkPreviewCrawler(r *http.Request) bool {
	for _, hdr := range []string{"Purpose", "Sec-Purpose", "X-Purpose", "X-Moz"} {
		v := strings.ToLower(strings.TrimSpace(r.Header.Get(hdr)))
		if strings.Contains(v, "prefetch") || strings.Contains(v, "preview") {
			return true
		}
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if ua == "" {
		return true
	}
	for _, tok := range previewBotTokens {
		if strings.Contains(ua, tok) {
			return true
		}
	}
	return false
}

func (h *PublicTrackableLinks) resolveDestinationPhone(ctx context.Context, link *domain.TrackableLink) (string, *int64) {
	fallback := util.NormalizePhone(link.FallbackPhone)
	if link.RoutingMode == domain.LinkRoutingManualPhone && fallback != "" {
		return fallback, nil
	}

	instances, _ := h.links.ListInstancesForPixel(ctx, link.PixelID)
	if link.RoutingMode == domain.LinkRoutingSpecificInstance && link.WhatsAppInstanceID != nil {
		for _, inst := range instances {
			if inst.ID == *link.WhatsAppInstanceID {
				if ph := extractPhoneFromOwnerJID(inst.OwnerJID); ph != "" {
					idCopy := inst.ID
					return ph, &idCopy
				}
			}
		}
	}

	openInsts := make([]domain.WhatsAppInstance, 0, len(instances))
	var anySavedPhone string
	var anySavedInstID *int64
	for _, inst := range instances {
		ph := extractPhoneFromOwnerJID(inst.OwnerJID)
		if ph == "" {
			continue
		}
		if anySavedPhone == "" {
			anySavedPhone = ph
			idCopy := inst.ID
			anySavedInstID = &idCopy
		}
		if isWhatsAppInstanceConnected(inst.ConnectionStatus) {
			openInsts = append(openInsts, inst)
		}
	}

	if len(openInsts) == 1 {
		idCopy := openInsts[0].ID
		return extractPhoneFromOwnerJID(openInsts[0].OwnerJID), &idCopy
	}
	if len(openInsts) > 1 {
		idx := 0
		if h.rdb != nil {
			rrKey := fmt.Sprintf("tf:wa_link_rr:%d", link.ID)
			if seq, err := h.rdb.Incr(ctx, rrKey).Result(); err == nil && seq > 0 {
				_ = h.rdb.Expire(ctx, rrKey, 30*24*time.Hour)
				idx = int((seq - 1) % int64(len(openInsts)))
			}
		}
		chosen := openInsts[idx]
		idCopy := chosen.ID
		return extractPhoneFromOwnerJID(chosen.OwnerJID), &idCopy
	}

	if fallback != "" {
		return fallback, nil
	}
	if anySavedPhone != "" {
		return anySavedPhone, anySavedInstID
	}
	return "", nil
}

func isWhatsAppInstanceConnected(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "OPEN", "CONNECTED", "ONLINE":
		return true
	default:
		return false
	}
}

func extractPhoneFromOwnerJID(ownerJID string) string {
	bare := strings.TrimSpace(ownerJID)
	if bare == "" {
		return ""
	}
	if at := strings.IndexByte(bare, '@'); at > 0 {
		bare = bare[:at]
	}
	if col := strings.IndexByte(bare, ':'); col > 0 {
		bare = bare[:col]
	}
	return util.NormalizePhone(rxJIDDigits.FindString(bare))
}

func firstNonEmptyTrim(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func inferSocialSource(ref string) string {
	r := strings.ToLower(ref)
	switch {
	case strings.Contains(r, "instagram.com"):
		return "instagram"
	case strings.Contains(r, "facebook.com") || strings.Contains(r, "fb.com"):
		return "facebook"
	case strings.Contains(r, "tiktok.com"):
		return "tiktok"
	case strings.Contains(r, "youtube.com") || strings.Contains(r, "youtu.be"):
		return "youtube"
	default:
		return "bio"
	}
}

func inferSocialMedium(ref, configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	if ref != "" {
		return "bio"
	}
	return "link_rastreavel"
}

func effectiveRequestHost(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xfh := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); xfh != "" {
		if idx := strings.IndexByte(xfh, ','); idx > 0 {
			xfh = strings.TrimSpace(xfh[:idx])
		}
		if xfh != "" {
			return xfh
		}
	}
	return strings.TrimSpace(r.Host)
}

func buildRequestFullURL(r *http.Request, fallback string) string {
	host := effectiveRequestHost(r)
	if r == nil || host == "" {
		return fallback
	}
	scheme := "https"
	if !isHTTPSRequest(r) && strings.HasPrefix(host, "localhost") {
		scheme = "http"
	}
	return scheme + "://" + host + r.URL.RequestURI()
}

func writeCrawlerPreviewHTML(w http.ResponseWriter, title string) {
	safeTitle := html.EscapeString(firstNonEmptyTrim(title, "Atendimento via WhatsApp"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(`<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><title>` + safeTitle + `</title><meta property="og:title" content="` + safeTitle + `"><meta property="og:description" content="Iniciar conversa no WhatsApp"><meta name="robots" content="noindex,nofollow"></head><body></body></html>`))
}

func writeUnavailableBridgeHTML(w http.ResponseWriter, title string) {
	safeTitle := html.EscapeString(firstNonEmptyTrim(title, "Atendimento"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + safeTitle + `</title><style>body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#121214;color:#f4f4f5;font-family:system-ui,sans-serif;text-align:center;padding:24px}</style></head><body><div><h2 style="margin:0 0 8px">Atendimento temporariamente indisponível</h2><p style="margin:0;color:#a1a1aa">Tente novamente em alguns instantes.</p></div></body></html>`))
}

func writeRedirectBridgeHTML(w http.ResponseWriter, p bridgeRenderParams) {
	msg := strings.TrimSpace(p.PrefillMessage)
	protoLine := "Protocolo: " + p.ProtocolCode
	fullText := protoLine
	if msg != "" {
		if strings.Contains(msg, protoLine) {
			fullText = msg
		} else {
			fullText = msg + "\n" + protoLine
		}
	}
	encText := url.QueryEscape(fullText)
	waMeURL := "https://wa.me/" + p.Phone + "?text=" + encText
	deepLink := "whatsapp://send?phone=" + p.Phone + "&text=" + encText
	waMeJSON, _ := json.Marshal(waMeURL)
	deepJSON, _ := json.Marshal(deepLink)
	safeTitle := html.EscapeString(firstNonEmptyTrim(p.Title, "Abrindo WhatsApp..."))

	fbqSnippet := ""
	if p.MetaBrowserPixelID != "" {
		pidJSON, _ := json.Marshal(p.MetaBrowserPixelID)
		extJSON, _ := json.Marshal(p.ExternalID)
		fbqSnippet = `!function(f,b,e,v,n,t,s){if(f.fbq)return;n=f.fbq=function(){n.callMethod?n.callMethod.apply(n,arguments):n.queue.push(arguments)};if(!f._fbq)f._fbq=n;n.push=n;n.loaded=!0;n.version='2.0';n.queue=[];t=b.createElement(e);t.async=!0;t.src=v;s=b.getElementsByTagName(e)[0];s.parentNode.insertBefore(t,s)}(window,document,'script','https://connect.facebook.net/en_US/fbevents.js');fbq('init',` + string(pidJSON) + `,{external_id:` + string(extJSON) + `});`
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	_, _ = w.Write([]byte(`<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="robots" content="noindex,nofollow"><title>` + safeTitle + `</title><style>*{box-sizing:border-box}body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#121214;color:#f4f4f5;font-family:system-ui,-apple-system,sans-serif;padding:24px;text-align:center}.box{max-width:340px;display:flex;flex-direction:column;align-items:center;gap:14px}.spin{width:34px;height:34px;border:3px solid rgba(37,211,102,.22);border-top-color:#25D366;border-radius:50%;animation:s .7s linear infinite}@keyframes s{to{transform:rotate(360deg)}}.btn{margin-top:6px;display:inline-block;padding:12px 22px;border-radius:999px;background:#25D366;color:#052e16;font-weight:600;text-decoration:none;font-size:15px}</style></head><body><div class="box"><div class="spin"></div><div style="font-size:16px;font-weight:600">Abrindo conversa no WhatsApp...</div><a id="wbtn" class="btn" href="` + html.EscapeString(waMeURL) + `">Continuar para o WhatsApp</a></div><script>` + fbqSnippet + `(function(){var deep=` + string(deepJSON) + `,web=` + string(waMeJSON) + `;var ua=navigator.userAgent||'';var isMob=/Android|iPhone|iPad|iPod/i.test(ua);setTimeout(function(){if(isMob){window.location.href=deep;setTimeout(function(){if(!document.hidden)window.location.replace(web);},650);}else{window.location.replace(web);}},120);})();</script></body></html>`))
}
