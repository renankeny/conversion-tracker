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

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type AdminPixels struct {
	repo      *repo.PixelRepo
	domains   *util.DomainSet // cache CORS — invalidado em mutações
	dnsTarget string          // host alvo esperado em CNAMEs dos clientes (ex: "track.trackfind.com.br")
}

func NewAdminPixels(r *repo.PixelRepo) *AdminPixels {
	return &AdminPixels{repo: r}
}

// SetDomainSet injeta o cache de domínios autorizados (CORS). Quando setado,
// é invalidado automaticamente em Create/Update/Delete pra evitar a janela
// de até 30s do refresh periódico.
func (h *AdminPixels) SetDomainSet(d *util.DomainSet) {
	h.domains = d
}

// refreshDomains recarrega o cache CORS a partir do banco. Erros geram log
// e seguem — o ticker periódico vai recuperar.
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

// SetDNSTarget configura o host alvo (registro de destino) usado para verificar
// se o tracking_host do cliente foi apontado corretamente via CNAME.
func (h *AdminPixels) SetDNSTarget(host string) {
	h.dnsTarget = strings.TrimSpace(host)
}

type pixelInput struct {
	Name          string                `json:"name"`
	Label         string                `json:"label"`
	PixelIDs      []string              `json:"pixel_ids"`
	AccessTokens  []string              `json:"access_tokens"`
	TestCodes     []string              `json:"test_codes"`
	Domains       []string              `json:"domains"`
	TrackerConfig *domain.TrackerConfig `json:"tracker_config"`
	Enabled       *bool                 `json:"enabled"`
	TrackingHost  *string               `json:"tracking_host"`
	AccountID     *int64                `json:"account_id"` // owners podem definir; members é ignorado

	// TikTok Events API (opcional - quando vazio, eventos só vão pra Meta)
	TikTokPixelIDs     []string `json:"tiktok_pixel_ids"`
	TikTokAccessTokens []string `json:"tiktok_access_tokens"`
	TikTokTestCodes    []string `json:"tiktok_test_codes"`
}

func (h *AdminPixels) List(w http.ResponseWriter, r *http.Request) {
	scope := TenantFromContext(r.Context())
	accountID := int64(0)
	if !scope.IsOwner {
		accountID = scope.AccountID
	}
	list, err := h.repo.ListByAccount(r.Context(), accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list error")
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
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	p, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && p.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	writeJSON(w, http.StatusOK, p.PublicView())
}

// Defaults devolve a TrackerConfig padrão pro frontend usar como template
// inicial quando criando um pixel novo.
func (h *AdminPixels) Defaults(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"tracker_config": domain.DefaultTrackerConfig(),
	})
}

func (h *AdminPixels) Create(w http.ResponseWriter, r *http.Request) {
	var in pixelInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
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
	scope := TenantFromContext(r.Context())
	override := int64(0)
	if in.AccountID != nil {
		override = *in.AccountID
	}
	p := &domain.Pixel{
		Name:          strings.TrimSpace(in.Name),
		Label:         in.Label,
		PixelIDs:      in.PixelIDs,
		AccessTokens:  in.AccessTokens,
		TestCodes:     padOrTrim(in.TestCodes, len(in.PixelIDs)),
		Domains:       cleanDomains(in.Domains),
		TrackerConfig: cfg,
		Enabled:       true,
		AccountID:     scope.EffectiveAccountID(override),

		TikTokPixelIDs:     in.TikTokPixelIDs,
		TikTokAccessTokens: in.TikTokAccessTokens,
		TikTokTestCodes:    padOrTrim(in.TikTokTestCodes, len(in.TikTokPixelIDs)),
	}
	if in.Enabled != nil {
		p.Enabled = *in.Enabled
	}
	if in.TrackingHost != nil {
		p.TrackingHost = normalizeHost(*in.TrackingHost)
	}
	if err := h.repo.Create(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, "create error: "+err.Error())
		return
	}
	h.refreshDomains(r.Context())
	writeJSON(w, http.StatusCreated, p.PublicView())
}

func (h *AdminPixels) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	var in pixelInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := validatePixelInput(in, false); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Name != "" {
		existing.Name = strings.TrimSpace(in.Name)
	}
	if in.Label != "" {
		existing.Label = in.Label
	}
	if in.PixelIDs != nil {
		existing.PixelIDs = in.PixelIDs
	}
	if in.AccessTokens != nil {
		existing.AccessTokens = in.AccessTokens
	}
	if in.TestCodes != nil {
		existing.TestCodes = padOrTrim(in.TestCodes, len(existing.PixelIDs))
	}
	if in.Domains != nil {
		existing.Domains = cleanDomains(in.Domains)
	}
	if in.TikTokPixelIDs != nil {
		existing.TikTokPixelIDs = in.TikTokPixelIDs
	}
	if in.TikTokAccessTokens != nil {
		existing.TikTokAccessTokens = in.TikTokAccessTokens
	}
	if in.TikTokTestCodes != nil {
		existing.TikTokTestCodes = padOrTrim(in.TikTokTestCodes, len(existing.TikTokPixelIDs))
	}
	if in.TrackerConfig != nil {
		existing.TrackerConfig = *in.TrackerConfig
	}
	if in.Enabled != nil {
		existing.Enabled = *in.Enabled
	}
	if in.TrackingHost != nil {
		existing.TrackingHost = normalizeHost(*in.TrackingHost)
	}
	if err := h.repo.Update(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "update error: "+err.Error())
		return
	}
	h.refreshDomains(r.Context())
	writeJSON(w, http.StatusOK, existing.PublicView())
}

// Delete remove o pixel. Por ser uma operação destrutiva no banco (perda
// de configuração e perda do vínculo histórico nos tracked_users), responde
// 409 se o body não tiver `confirm: true`. O frontend deve fazer dupla
// confirmação antes do request final.
func (h *AdminPixels) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "load error")
		return
	}
	scope := TenantFromContext(r.Context())
	if !scope.IsOwner && existing.AccountID != scope.AccountID {
		writeError(w, http.StatusForbidden, "sem acesso")
		return
	}
	var body struct {
		Confirm bool `json:"confirm"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if !body.Confirm {
		writeError(w, http.StatusConflict, "destructive operation requires {\"confirm\": true} in body")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "delete error")
		return
	}
	h.refreshDomains(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

func validatePixelInput(in pixelInput, requireAll bool) error {
	if requireAll {
		if strings.TrimSpace(in.Name) == "" {
			return errors.New("name required")
		}
		if len(in.PixelIDs) == 0 {
			return errors.New("at least one pixel_id required")
		}
		if len(in.AccessTokens) != len(in.PixelIDs) {
			return errors.New("access_tokens length must match pixel_ids")
		}
	}
	if in.PixelIDs != nil && in.AccessTokens != nil &&
		len(in.AccessTokens) != len(in.PixelIDs) {
		return errors.New("access_tokens length must match pixel_ids")
	}
	// TikTok: se algum dos dois for informado, ambos devem ter o mesmo tamanho.
	if in.TikTokPixelIDs != nil && in.TikTokAccessTokens != nil &&
		len(in.TikTokAccessTokens) != len(in.TikTokPixelIDs) {
		return errors.New("tiktok_access_tokens length must match tiktok_pixel_ids")
	}
	return nil
}

func padOrTrim(s []string, n int) []string {
	out := make([]string, n)
	for i := 0; i < n && i < len(s); i++ {
		out[i] = s[i]
	}
	return out
}

// cleanDomains normaliza domínios (lowercase, trim, sem duplicatas e sem strings vazias).
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

// normalizeHost limpa um host: remove scheme, path, espaço e força lowercase.
// Aceita "https://tracking.cliente.com/" e devolve "tracking.cliente.com".
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
