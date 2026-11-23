package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/renankeny/conversion-tracker/backend/internal/cache"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/geoip"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/traffic"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

var rxJIDDigits = regexp.MustCompile(`\d+`)

type PublicTrackableLinks struct {
	links              *repo.TrackableLinkRepo
	pixels             *repo.PixelRepo
	users              *repo.UserRepo
	geo                *geoip.Lookup
	queue              *queue.Enqueuer
	urls               *util.URLMapper
	rdb                *redis.Client
	publicAPIURL       string
	trackingTargetHost string
}

func NewPublicTrackableLinks(
	links *repo.TrackableLinkRepo,
	pixels *repo.PixelRepo,
	users *repo.UserRepo,
	geo *geoip.Lookup,
	q *queue.Enqueuer,
	urls *util.URLMapper,
	rdb *redis.Client,
	publicAPIURL, trackingTargetHost string,
) *PublicTrackableLinks {
	return &PublicTrackableLinks{
		links:              links,
		pixels:             pixels,
		users:              users,
		geo:                geo,
		queue:              q,
		urls:               urls,
		rdb:                rdb,
		publicAPIURL:       publicAPIURL,
		trackingTargetHost: strings.TrimSpace(trackingTargetHost),
	}
}

func (h *PublicTrackableLinks) ServeRedirect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slug := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "slug")))
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	link, err := h.links.FindBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		slog.Error("find trackable link by slug", "slug", slug, "error", err)
		http.Error(w, "erro temporário", http.StatusInternalServerError)
		return
	}
	if !link.Enabled {
		http.NotFound(w, r)
		return
	}

	pixel, err := h.pixels.FindByID(ctx, link.PixelID)
	if err != nil || pixel == nil || !pixel.Enabled {
		http.NotFound(w, r)
		return
	}

	if IsLinkPreviewCrawler(r) {
		writeCrawlerPreviewHTML(w, link.Name)
		return
	}

	destPhone, resolvedInstID := h.resolveDestinationPhone(ctx, link)
	if destPhone == "" {
		writeUnavailableBridgeHTML(w, link.Name)
		return
	}

	ip := clientIP(r)
	ua := r.Header.Get("User-Agent")
	ref := strings.TrimSpace(r.Header.Get("Referer"))

	var country, state, city, postal string
	if h.geo != nil {
		if g, gerr := h.geo.Resolve(ip); gerr == nil {
			country = strings.ToLower(g.Country)
			state = strings.ToLower(g.State)
			city = util.NormalizeCity(g.City)
			postal = g.PostalCode
		}
	}

	userID, idWpp, fbp, fbc, ttp, ga4CID := extractOrSynthesizeBridgeIDs(r)
	protoCode := userID
	q := r.URL.Query()

	utmSource := firstNonEmptyTrim(q.Get("utm_source"), link.UTMSource, inferSocialSource(ref))
	utmMedium := firstNonEmptyTrim(q.Get("utm_medium"), link.UTMMedium, inferSocialMedium(ref, link.UTMMedium))
	utmCampaign := firstNonEmptyTrim(q.Get("utm_campaign"), link.UTMCampaign, slugifyTrackableLink(link.Name), link.Slug)
	utmContent := firstNonEmptyTrim(q.Get("utm_content"), link.UTMContent)
	utmTerm := firstNonEmptyTrim(q.Get("utm_term"), link.UTMTerm)

	entryURL := buildRequestFullURL(r, link.BuildPublicURL(h.publicAPIURL))
	if h.urls != nil {
		entryURL = h.urls.ApplyForAccount(pixel.AccountID, entryURL)
	}

	trafficRes := traffic.Classify(ua, ref, false)
	rulesJSON, _ := json.Marshal(trafficRes.Rules)
	isBot := trafficRes.Type == "bot"
	linkIDCopy := link.ID

	var trackedUserID *int64
	tu := &domain.TrackedUser{
		ExternalID:       userID,
		IDWpp:            idWpp,
		PixelName:        pixel.Name,
		AccountID:        pixel.AccountID,
		TrackableLinkID:  &linkIDCopy,
		FBP:              fbp,
		FBC:              fbc,
		GCLID:            strings.TrimSpace(q.Get("gclid")),
		WBraid:           strings.TrimSpace(q.Get("wbraid")),
		GBraid:           strings.TrimSpace(q.Get("gbraid")),
		TTClid:           strings.TrimSpace(q.Get("ttclid")),
		TTP:              ttp,
		GA4ClientID:      ga4CID,
		ClientIP:         ip,
		ClientUserAgent:  ua,
		Country:          country,
		State:            state,
		City:             city,
		PostalCode:       postal,
		URL:              entryURL,
		FirstURL:         entryURL,
		FirstUTMSource:   util.StripUserIDFromUTM(utmSource),
		FirstUTMMedium:   utmMedium,
		FirstUTMCampaign: utmCampaign,
		FirstUTMContent:  utmContent,
		FirstUTMTerm:     utmTerm,
		LastUTMSource:    util.StripUserIDFromUTM(utmSource),
		LastUTMMedium:    utmMedium,
		LastUTMCampaign:  utmCampaign,
		LastUTMContent:   utmContent,
		LastUTMTerm:      utmTerm,
		PageViewCount:    1,
		Referrer:         ref,
		TrafficType:      trafficRes.Type,
		TrafficScore:     trafficRes.Score,
		TrafficCategory:  trafficRes.Category,
		TrafficRules:     json.RawMessage(rulesJSON),
	}
	if !isBot && h.users != nil {
		if uerr := h.users.Upsert(ctx, tu); uerr == nil && tu.ID > 0 {
			trackedUserID = &tu.ID
		}
	}

	setBridgeTrackingCookies(w, r, h.trackingTargetHost, pixel.TrackingHost, sendRequest{
		UserID: userID, FBP: fbp, FBC: fbc, TTP: ttp,
	})

	allowedChans := link.AllowedChannels()
	fireOnClick := !isBot && link.SendEventsEnabled && link.EventTriggerMode == domain.LinkTriggerOnClick && len(allowedChans) > 0

	click := &domain.TrackableLinkClick{
		AccountID:          pixel.AccountID,
		PixelID:            pixel.ID,
		LinkID:             link.ID,
		TrackedUserID:      trackedUserID,
		ResolvedInstanceID: resolvedInstID,
		ProtocolCode:       protoCode,
		ClientIP:           ip,
		IsBot:              isBot,
		ClickEventSent:     fireOnClick,
	}
	_ = h.links.RecordClick(ctx, click)

	if fireOnClick && h.queue != nil {
		evID := fmt.Sprintf("walnk_clk_%d", click.ID)
		if click.ID == 0 {
			evID = "walnk_clk_" + protoCode
		}
		_ = h.queue.EnqueueSendEvent(ctx, domain.SendEventInput{
			PixelName: pixel.Name, EventName: link.EventName, EventID: evID, EventTime: time.Now().Unix(),
			SourceURL: entryURL, Gateway: "whatsapp_link", ActionSource: "website",
			ExternalID: userID, IDWpp: idWpp, ClientIP: ip, ClientUserAgent: ua,
			Country: country, State: state, City: city, PostalCode: postal,
			FBP: fbp, FBC: fbc, GCLID: tu.GCLID, WBraid: tu.WBraid, GBraid: tu.GBraid,
			TTClid: tu.TTClid, TTP: ttp, Referrer: ref, GA4ClientID: ga4CID,
			AllowedChannels: allowedChans,
		})
	}

	if h.rdb != nil && !isBot {
		cache.BumpDashboardVersion(ctx, h.rdb, pixel.AccountID)
	}

	metaBrowserPixelID := ""
	if link.SendEventsEnabled && link.SendMeta && len(pixel.PixelIDs) > 0 {
		metaBrowserPixelID = strings.TrimSpace(pixel.TrackerConfig.MetaInitPixelID)
		if metaBrowserPixelID == "" {
			metaBrowserPixelID = strings.TrimSpace(pixel.PixelIDs[0])
		}
	}
	writeRedirectBridgeHTML(w, bridgeRenderParams{
		Title:              link.Name,
		Phone:              destPhone,
		PrefillMessage:     link.PrefillMessage,
		ProtocolCode:       protoCode,
		MetaBrowserPixelID: metaBrowserPixelID,
		ExternalID:         userID,
	})
}
