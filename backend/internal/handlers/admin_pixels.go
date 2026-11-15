package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type AdminPixels struct {
	repo      *repo.PixelRepo
	pool      *pgxpool.Pool
	gads      *repo.GoogleAdsAccountRepo
	metaAccs  *repo.MetaAccountRepo
	domains   *util.DomainSet // cache CORS invalidado em mutações
	dnsTarget string          // host alvo esperado em CNAMEs dos clientes
}

func NewAdminPixels(r *repo.PixelRepo) *AdminPixels {
	return &AdminPixels{repo: r}
}

func (h *AdminPixels) SetSecurityDeps(pool *pgxpool.Pool, gads *repo.GoogleAdsAccountRepo) {
	h.pool = pool
	h.gads = gads
}

func (h *AdminPixels) SetMetaAccountsRepo(metaAccs *repo.MetaAccountRepo) {
	h.metaAccs = metaAccs
}

func (h *AdminPixels) SetDomainSet(d *util.DomainSet) {
	h.domains = d
}

// refreshDomains recarrega o cache CORS a partir do banco.
func (h *AdminPixels) refreshDomains(ctx context.Context) {
	if h.domains == nil {
		return
	}
	list, err := h.repo.ActiveDomains(ctx)
	if err != nil {
		slog.Warn("refresh allowed_domains", "err", err)
		return
	}
	h.domains.Set(list)
}

func (h *AdminPixels) SetDNSTarget(host string) {
	h.dnsTarget = strings.TrimSpace(host)
}

type pixelInput struct {
	Name          string                `json:"name"`
	Label         *string               `json:"label"`
	PixelIDs      []string              `json:"pixel_ids"`
	AccessTokens  []string              `json:"access_tokens"`
	TestCodes     []string              `json:"test_codes"`
	Domains       []string              `json:"domains"`
	TrackerConfig *domain.TrackerConfig `json:"tracker_config"`
	Enabled       *bool                 `json:"enabled"`
	TrackingHost  *string               `json:"tracking_host"`
	AccountID     *int64                `json:"account_id"`

	// TikTok Events API (opcional)
	TikTokPixelIDs     []string `json:"tiktok_pixel_ids"`
	TikTokAccessTokens []string `json:"tiktok_access_tokens"`
	TikTokTestCodes    []string `json:"tiktok_test_codes"`

	// Google Ads (aceita 0 ou null para desvincular + Customer ID específico do pixel)
	GoogleAdsAccountID       *int64  `json:"google_ads_account_id"`
	GoogleAdsCustomerID      *string `json:"google_ads_customer_id"`
	GoogleAdsCustomerName    *string `json:"google_ads_customer_name"`
	GoogleAdsLoginCustomerID *string `json:"google_ads_login_customer_id"`

	// GA4 Measurement Protocol
	GA4MeasurementID *string `json:"ga4_measurement_id"`
	GA4APISecret     *string `json:"ga4_api_secret"`

	// Meta Permanent Token, Ad Account & OAuth Account
	MetaPermanentToken *string `json:"meta_permanent_token"`
	MetaAdAccountID    *string `json:"meta_ad_account_id"`
	MetaAccountID      *int64  `json:"meta_account_id"`

	// Agrupamento de Conversões (Order Bump & Upsell)
	AggregationEnabled       *bool `json:"aggregation_enabled"`
	AggregationWindowMinutes *int  `json:"aggregation_window_minutes"`
}

func (h *AdminPixels) List(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	list, err := h.repo.ListByAccount(r.Context(), scope.AccountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar pixels")
		return
	}
	out := make([]map[string]any, len(list))
	for i, p := range list {
		out[i] = p.PublicView()
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *AdminPixels) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ID inválido")
		return
	}
	p, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "pixel não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao carregar pixel")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && p.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	writeJSON(w, http.StatusOK, p.PublicView())
}

func (h *AdminPixels) Defaults(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"tracker_config": domain.DefaultTrackerConfig(),
	})
}

func (h *AdminPixels) Create(w http.ResponseWriter, r *http.Request) {
	if !auth.CanEditPixels(r.Context()) {
		writeError(w, http.StatusForbidden, "você não tem permissão para editar pixels")
		return
	}
	var in pixelInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	if err := validatePixelInput(in, true); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg := domain.DefaultTrackerConfig()
	if in.TrackerConfig != nil {
		cfg = *in.TrackerConfig
	}
	targetAccountID, err := ResolveTargetAccountID(r.Context(), h.pool, in.AccountID)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	cleanMetaIDs := cleanStringSlice(in.PixelIDs)
	cleanTikTokIDs := cleanStringSlice(in.TikTokPixelIDs)
	p := &domain.Pixel{
		Name:               strings.TrimSpace(in.Name),
		Label:              strings.TrimSpace(derefString(in.Label)),
		PixelIDs:           cleanMetaIDs,
		AccessTokens:       padOrTrim(in.AccessTokens, len(cleanMetaIDs)),
		TestCodes:          []string{},
		Domains:            cleanDomains(in.Domains),
		TrackerConfig:      cfg,
		Enabled:            true,
		AccountID:          targetAccountID,
		TikTokPixelIDs:     cleanTikTokIDs,
		TikTokAccessTokens: padOrTrim(in.TikTokAccessTokens, len(cleanTikTokIDs)),
		TikTokTestCodes:    []string{},
	}
	if in.GoogleAdsAccountID != nil {
		p.GoogleAdsAccountID = nilIfZeroInt64(*in.GoogleAdsAccountID)
	}
	if in.MetaAccountID != nil {
		p.MetaAccountID = nilIfZeroInt64(*in.MetaAccountID)
	}
	if err := RequireAccountGoogleAds(r.Context(), h.gads, targetAccountID, p.GoogleAdsAccountID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if err := RequireAccountMetaAds(r.Context(), h.metaAccs, targetAccountID, p.MetaAccountID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	applyOptionalPixelFields(p, in)
	syncPixelGoogleAdsDefaults(r.Context(), h.gads, p)
	if err := h.repo.Create(r.Context(), p); err != nil {
		if isDuplicateKey(err) {
			writeError(w, http.StatusConflict, "já existe um pixel com este nome (slug); escolha outro identificador")
			return
		}
		if isForeignKeyViolation(err) {
			writeError(w, http.StatusBadRequest, "sua conta associada não existe mais; faça login novamente ou contate o suporte")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao criar pixel: "+err.Error())
		return
	}
	h.refreshDomains(r.Context())
	h.repo.InvalidateCache(r.Context(), p.Name, p.PublicToken)
	h.repo.BumpDashboardCache(r.Context(), p.AccountID)
	writeJSON(w, http.StatusCreated, p.PublicView())
}
