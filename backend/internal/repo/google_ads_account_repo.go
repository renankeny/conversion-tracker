package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

// GoogleAdsAccountRepo persiste contas Google Ads conectadas via OAuth.
type GoogleAdsAccountRepo struct {
	pool *pgxpool.Pool
}

func NewGoogleAdsAccountRepo(pool *pgxpool.Pool) *GoogleAdsAccountRepo {
	return &GoogleAdsAccountRepo{pool: pool}
}

const googleAdsAccountColumns = `id, account_id, name, customer_id, login_customer_id,
		conversion_action_id, conversion_actions, refresh_token, oauth_email, oauth_picture_url, enabled,
		created_at, updated_at`

func (r *GoogleAdsAccountRepo) FindByID(ctx context.Context, id int64) (*domain.GoogleAdsAccount, error) {
	q := `SELECT ` + googleAdsAccountColumns + ` FROM google_ads_accounts WHERE id = $1`
	return scanGoogleAdsAccount(r.pool.QueryRow(ctx, q, id))
}

// List retorna todas as contas (uso owner). Use ListByAccount para filtrar
// por conta TrackFind (uso member).
func (r *GoogleAdsAccountRepo) List(ctx context.Context) ([]domain.GoogleAdsAccount, error) {
	q := `SELECT ` + googleAdsAccountColumns + ` FROM google_ads_accounts ORDER BY id DESC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.GoogleAdsAccount, 0)
	for rows.Next() {
		a, err := scanGoogleAdsAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// ListByAccount filtra por conta TrackFind (multi-tenant).
func (r *GoogleAdsAccountRepo) ListByAccount(ctx context.Context, accountID int64) ([]domain.GoogleAdsAccount, error) {
	if accountID == 0 {
		return r.List(ctx)
	}
	q := `SELECT ` + googleAdsAccountColumns + `
	      FROM google_ads_accounts
	      WHERE account_id = $1
	      ORDER BY id DESC`
	rows, err := r.pool.Query(ctx, q, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.GoogleAdsAccount, 0)
	for rows.Next() {
		a, err := scanGoogleAdsAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *GoogleAdsAccountRepo) Create(ctx context.Context, a *domain.GoogleAdsAccount) error {
	if a.AccountID == 0 {
		a.AccountID = 1
	}
	if a.ConversionActions == nil {
		a.ConversionActions = []domain.GoogleAdsConversionAction{}
	}
	actionsJSON, err := json.Marshal(a.ConversionActions)
	if err != nil {
		return err
	}
	encTok, err := util.Encrypt(a.RefreshToken)
	if err != nil {
		return err
	}
	const q = `INSERT INTO google_ads_accounts
	    (account_id, name, customer_id, login_customer_id, conversion_action_id,
	     conversion_actions, refresh_token, oauth_email, oauth_picture_url, enabled)
	    VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10)
	    RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, q,
		a.AccountID, a.Name, a.CustomerID, a.LoginCustomerID, a.ConversionActionID,
		actionsJSON, encTok, a.OAuthEmail, a.OAuthPictureURL, a.Enabled,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// Update atualiza campos editaveis pelo painel (nome + IDs + actions + enabled).
// refresh_token e oauth_email sao gerenciados pelo fluxo OAuth via
// SetTokens / ClearTokens.
func (r *GoogleAdsAccountRepo) Update(ctx context.Context, a *domain.GoogleAdsAccount) error {
	if a.ConversionActions == nil {
		a.ConversionActions = []domain.GoogleAdsConversionAction{}
	}
	actionsJSON, err := json.Marshal(a.ConversionActions)
	if err != nil {
		return err
	}
	const q = `UPDATE google_ads_accounts
	           SET name = $1,
	               customer_id = $2,
	               login_customer_id = $3,
	               conversion_actions = $4::jsonb,
	               enabled = $5,
	               updated_at = NOW()
	           WHERE id = $6
	           RETURNING updated_at`
	err = r.pool.QueryRow(ctx, q,
		a.Name, a.CustomerID, a.LoginCustomerID, actionsJSON, a.Enabled, a.ID,
	).Scan(&a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *GoogleAdsAccountRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM google_ads_accounts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GoogleAdsAccountRepo) SetTokens(ctx context.Context, id int64, refreshToken, email, pictureURL string) error {
	encTok, err := util.Encrypt(refreshToken)
	if err != nil {
		return err
	}
	const q = `UPDATE google_ads_accounts
	           SET refresh_token = $2,
	               oauth_email = $3,
	               oauth_picture_url = $4,
	               updated_at = NOW()
	           WHERE id = $1`
	tag, err := r.pool.Exec(ctx, q, id, encTok, email, pictureURL)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GoogleAdsAccountRepo) UpdateOAuthPicture(ctx context.Context, id int64, pictureURL string) error {
	if pictureURL == "" {
		return nil
	}
	_, err := r.pool.Exec(ctx, `UPDATE google_ads_accounts SET oauth_picture_url = $2 WHERE id = $1`, id, pictureURL)
	return err
}

func (r *GoogleAdsAccountRepo) ClearTokens(ctx context.Context, id int64) error {
	const q = `UPDATE google_ads_accounts
	           SET refresh_token = '',
	               oauth_email = '',
	               oauth_picture_url = '',
	               updated_at = NOW()
	           WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id)
	return err
}

func scanGoogleAdsAccount(row rowScanner) (*domain.GoogleAdsAccount, error) {
	var a domain.GoogleAdsAccount
	var actionsJSON []byte
	err := row.Scan(
		&a.ID, &a.AccountID, &a.Name, &a.CustomerID, &a.LoginCustomerID,
		&a.ConversionActionID, &actionsJSON, &a.RefreshToken, &a.OAuthEmail, &a.OAuthPictureURL, &a.Enabled,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.RefreshToken = util.Decrypt(a.RefreshToken)
	if len(actionsJSON) > 0 {
		if jerr := json.Unmarshal(actionsJSON, &a.ConversionActions); jerr != nil {
			return nil, jerr
		}
	}
	if a.ConversionActions == nil {
		a.ConversionActions = []domain.GoogleAdsConversionAction{}
	}
	return &a, nil
}
