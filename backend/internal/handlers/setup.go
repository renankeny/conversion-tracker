package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// SetupHandler gerencia o wizard de primeiro acesso (onboarding do Super Admin,
// empresa principal e identidade visual Whitelabel).
type SetupHandler struct {
	pool         *pgxpool.Pool
	jwt          *auth.JWT
	brandingRepo *repo.BrandingRepo
	calibrator   *util.ShardCalibrator
}

func NewSetupHandler(pool *pgxpool.Pool, jwt *auth.JWT, brandingRepo *repo.BrandingRepo) *SetupHandler {
	return &SetupHandler{
		pool:         pool,
		jwt:          jwt,
		brandingRepo: brandingRepo,
	}
}

func (h *SetupHandler) SetCalibrator(c *util.ShardCalibrator) {
	h.calibrator = c
}

type setupInitReq struct {
	AdminName     string                  `json:"admin_name"`
	AdminEmail    string                  `json:"admin_email"`
	AdminPhone    string                  `json:"admin_phone"`
	AdminPassword string                  `json:"admin_password"`
	AccountName   string                  `json:"account_name"`
	Branding      domain.PlatformBranding `json:"branding"`
}

// Status GET /api/v1/setup/status
// Retorna needs_setup=true quando não existe nenhum usuário em admin_users,
// e sincroniza a fatia temporal diária (00:01 / entrada no aplicativo).
func (h *SetupHandler) Status(w http.ResponseWriter, r *http.Request) {
	var count int
	if err := h.pool.QueryRow(r.Context(), `SELECT COUNT(*) FROM admin_users`).Scan(&count); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao verificar status de instalação")
		return
	}
	var notice *util.ShardServiceNotice
	if h.calibrator != nil && count > 0 {
		_, notice = h.calibrator.VerifyEpochSlice(r.Context())
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"needs_setup":    count == 0,
		"service_notice": notice,
	})
}

// VerifyEmail POST /api/v1/setup/verify-email
// Valida na hora o e-mail informado na etapa 1 do Setup.
func (h *SetupHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload inválido")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "Informe um e-mail válido (o mesmo utilizado na compra do sistema).")
		return
	}
	if h.calibrator != nil {
		if err := h.calibrator.CalibrateIdentity(r.Context(), email); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"valid": true})
}

// Init POST /api/v1/setup/init
// Executa a configuração inicial atômica (somente se admin_users estiver vazio).
func (h *SetupHandler) Init(w http.ResponseWriter, r *http.Request) {
	var req setupInitReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "payload inválido")
		return
	}

	req.AdminName = strings.TrimSpace(req.AdminName)
	req.AdminEmail = strings.ToLower(strings.TrimSpace(req.AdminEmail))
	req.AdminPhone = strings.TrimSpace(req.AdminPhone)
	req.AccountName = strings.TrimSpace(req.AccountName)

	if req.AdminName == "" {
		writeError(w, http.StatusBadRequest, "nome do administrador é obrigatório")
		return
	}
	if req.AdminEmail == "" || !strings.Contains(req.AdminEmail, "@") {
		writeError(w, http.StatusBadRequest, "e-mail válido é obrigatório")
		return
	}
	if len(req.AdminPassword) < 6 {
		writeError(w, http.StatusBadRequest, "a senha deve ter pelo menos 6 caracteres")
		return
	}
	if req.AccountName == "" {
		req.AccountName = "Conta Principal"
	}

	if h.calibrator != nil {
		if err := h.calibrator.CalibrateIdentity(r.Context(), req.AdminEmail); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	hash, err := auth.HashPassword(req.AdminPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao processar senha")
		return
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao iniciar transação")
		return
	}
	defer tx.Rollback(r.Context())

	var existing int
	if err := tx.QueryRow(r.Context(), `SELECT COUNT(*) FROM admin_users`).Scan(&existing); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao validar estado do sistema")
		return
	}
	if existing > 0 {
		writeError(w, http.StatusConflict, "o sistema já foi configurado anteriormente")
		return
	}

	var accountID int64
	if err := tx.QueryRow(r.Context(),
		`UPDATE accounts SET name = $1, enabled = TRUE, updated_at = NOW()
		 WHERE id = (SELECT id FROM accounts ORDER BY id ASC LIMIT 1) RETURNING id`,
		req.AccountName,
	).Scan(&accountID); err != nil {
		if err := tx.QueryRow(r.Context(),
			`INSERT INTO accounts (name, slug, enabled) VALUES ($1, 'default', TRUE) RETURNING id`,
			req.AccountName,
		).Scan(&accountID); err != nil {
			slog.Error("setup create account", "err", err)
			writeError(w, http.StatusInternalServerError, "erro ao criar empresa principal")
			return
		}
	}

	var adminID int64
	if err := tx.QueryRow(r.Context(),
		`INSERT INTO admin_users (email, password_hash, name, phone, role, account_id, permissions)
		 VALUES ($1, $2, $3, $4, 'superadmin', $5, '{}'::jsonb)
		 RETURNING id`,
		req.AdminEmail, hash, req.AdminName, req.AdminPhone, accountID,
	).Scan(&adminID); err != nil {
		slog.Error("setup create superadmin", "err", err)
		writeError(w, http.StatusInternalServerError, "erro ao criar super admin")
		return
	}

	if _, err := tx.Exec(r.Context(), `UPDATE accounts SET owner_id = $1 WHERE id = $2`, adminID, accountID); err != nil {
		slog.Error("setup set account owner", "err", err)
	}

	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("setup commit", "err", err)
		writeError(w, http.StatusInternalServerError, "erro ao finalizar configuração inicial")
		return
	}

	savedBranding := h.saveInitialBranding(r, req.Branding)
	tok, err := h.jwt.IssueWithDuration(adminID, req.AdminEmail, "superadmin", accountID, nil, map[string]any{}, 30*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao emitir token de sessão")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"access_token": tok,
		"user": map[string]any{
			"id":          adminID,
			"email":       req.AdminEmail,
			"name":        req.AdminName,
			"phone":       req.AdminPhone,
			"role":        "superadmin",
			"account_id":  accountID,
			"preferences": map[string]any{},
		},
		"branding": savedBranding,
	})
}

func (h *SetupHandler) saveInitialBranding(r *http.Request, in domain.PlatformBranding) domain.PlatformBranding {
	def := domain.DefaultPlatformBranding()
	if h.brandingRepo == nil {
		return def
	}
	in.PlatformName = strings.TrimSpace(in.PlatformName)
	if in.PlatformName == "" {
		in.PlatformName = def.PlatformName
	}
	in.Tagline = strings.TrimSpace(in.Tagline)
	if in.Tagline == "" {
		in.Tagline = def.Tagline
	}
	if in.LogoDisplayMode == "" {
		in.LogoDisplayMode = def.LogoDisplayMode
	}
	if !validatePalette(&in.ThemeConfig.Dark) || in.ThemeConfig.Dark.Brand == "" {
		in.ThemeConfig.Dark = def.ThemeConfig.Dark
	}
	if !validatePalette(&in.ThemeConfig.Light) || in.ThemeConfig.Light.Brand == "" {
		in.ThemeConfig.Light = def.ThemeConfig.Light
	}
	if len(in.LogoDarkURL) > maxDataURLLength || !SafeLogoOrFaviconURL(in.LogoDarkURL) {
		in.LogoDarkURL = ""
	}
	if len(in.LogoLightURL) > maxDataURLLength || !SafeLogoOrFaviconURL(in.LogoLightURL) {
		in.LogoLightURL = ""
	}
	if len(in.FaviconURL) > maxDataURLLength || !SafeLogoOrFaviconURL(in.FaviconURL) {
		in.FaviconURL = ""
	}
	saved, err := h.brandingRepo.Upsert(r.Context(), in)
	if err != nil {
		return def
	}
	return saved
}
