package handlers

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

var validLinkSlugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,78}$`)

type trackableLinkInput struct {
	Name               string `json:"name"`
	RoutingMode        string `json:"routing_mode"`
	WhatsAppInstanceID *int64 `json:"whatsapp_instance_id"`
	FallbackPhone      string `json:"fallback_phone"`
	PrefillMessage     string `json:"prefill_message"`
	UTMSource          string `json:"utm_source"`
	UTMMedium          string `json:"utm_medium"`
	UTMCampaign        string `json:"utm_campaign"`
	UTMContent         string `json:"utm_content"`
	UTMTerm            string `json:"utm_term"`
	SendEventsEnabled  *bool  `json:"send_events_enabled"`
	EventTriggerMode   string `json:"event_trigger_mode"`
	EventName          string `json:"event_name"`
	SendMeta           *bool  `json:"send_meta"`
	SendTikTok         *bool  `json:"send_tiktok"`
	SendGoogleAds      *bool  `json:"send_google_ads"`
	SendGA4            *bool  `json:"send_ga4"`
	Enabled            *bool  `json:"enabled"`
}

func (h *AdminTrackableLinks) buildAndValidateInput(
	ctx context.Context,
	pixel *domain.Pixel,
	existing *domain.TrackableLink,
	in trackableLinkInput,
) (*domain.TrackableLink, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, errors.New("informe o nome do link (ex: Bio Instagram)")
	}

	slug := ""
	if existing != nil && validLinkSlugRe.MatchString(existing.Slug) {
		slug = existing.Slug
	} else {
		slug = domain.GenerateTrackableLinkSlug(pixel.AccountID)
	}

	routingMode := domain.NormalizeLinkRoutingMode(in.RoutingMode)
	var instID *int64
	if in.WhatsAppInstanceID != nil && *in.WhatsAppInstanceID > 0 {
		inst, err := h.wa.GetInstance(ctx, pixel.AccountID, *in.WhatsAppInstanceID)
		if err != nil || inst == nil || inst.PixelID != pixel.ID {
			return nil, errors.New("a instância WhatsApp selecionada não pertence a este pixel")
		}
		idCopy := inst.ID
		instID = &idCopy
	} else if routingMode == domain.LinkRoutingSpecificInstance {
		return nil, errors.New("selecione qual instância WhatsApp receberá os leads ou use o modo automático/rodízio")
	}

	fallbackPhone := ""
	if strings.TrimSpace(in.FallbackPhone) != "" {
		fallbackPhone = util.NormalizePhone(in.FallbackPhone)
		if len(fallbackPhone) < 10 {
			return nil, errors.New("telefone reserva inválido: informe DDD + número com código do país (ex: 5511999998888)")
		}
	}
	if routingMode == domain.LinkRoutingManualPhone && fallbackPhone == "" {
		return nil, errors.New("informe o telefone de destino para o modo de número fixo")
	}

	prefill := strings.TrimSpace(in.PrefillMessage)
	if prefill == "" {
		prefill = "Olá! Gostaria de mais informações."
	}

	out := &domain.TrackableLink{
		AccountID:          pixel.AccountID,
		PixelID:            pixel.ID,
		Slug:               slug,
		Name:               name,
		RoutingMode:        routingMode,
		WhatsAppInstanceID: instID,
		FallbackPhone:      fallbackPhone,
		PrefillMessage:     prefill,
		UTMSource:          strings.TrimSpace(in.UTMSource),
		UTMMedium:          strings.TrimSpace(in.UTMMedium),
		UTMCampaign:        strings.TrimSpace(in.UTMCampaign),
		UTMContent:         strings.TrimSpace(in.UTMContent),
		UTMTerm:            strings.TrimSpace(in.UTMTerm),
		SendEventsEnabled:  boolOrDefault(in.SendEventsEnabled, true),
		EventTriggerMode:   domain.NormalizeLinkTriggerMode(in.EventTriggerMode),
		EventName:          domain.NormalizeLinkEventName(in.EventName),
		SendMeta:           boolOrDefault(in.SendMeta, true),
		SendTikTok:         boolOrDefault(in.SendTikTok, true),
		SendGoogleAds:      boolOrDefault(in.SendGoogleAds, true),
		SendGA4:            boolOrDefault(in.SendGA4, true),
		Enabled:            boolOrDefault(in.Enabled, true),
	}
	if existing != nil {
		out.ID = existing.ID
		out.CreatedAt = existing.CreatedAt
	}
	return out, nil
}

func boolOrDefault(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

func slugifyTrackableLink(raw string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	clean, _, err := transform.String(t, strings.ToLower(strings.TrimSpace(raw)))
	if err != nil {
		clean = strings.ToLower(strings.TrimSpace(raw))
	}
	var b strings.Builder
	lastDash := false
	for i := 0; i < len(clean); i++ {
		c := clean[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteByte(c)
			lastDash = false
		} else if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	res := strings.Trim(b.String(), "-_")
	if len(res) > 78 {
		res = strings.Trim(res[:78], "-_")
	}
	return res
}
