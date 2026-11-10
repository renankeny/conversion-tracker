package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

type MetaAccountRepo struct {
	pool *pgxpool.Pool
}

func NewMetaAccountRepo(pool *pgxpool.Pool) *MetaAccountRepo {
	return &MetaAccountRepo{pool: pool}
}

const metaAccountCols = `id, account_id, name, vault_token_encrypted,
	oauth_user_id, oauth_name, oauth_email, oauth_picture_url, token_expires_at,
	ad_account_id, ad_account_name, ad_account_picture_url,
	default_page_id, default_page_name, default_page_picture_url,
	enabled, created_at, updated_at`

func scanMetaAccount(row pgx.Row) (*domain.MetaAccount, error) {
	var a domain.MetaAccount
	err := row.Scan(
		&a.ID, &a.AccountID, &a.Name, &a.VaultTokenEncrypted,
		&a.OAuthUserID, &a.OAuthName, &a.OAuthEmail, &a.OAuthPictureURL, &a.TokenExpiresAt,
		&a.AdAccountID, &a.AdAccountName, &a.AdAccountPictureURL,
		&a.DefaultPageID, &a.DefaultPageName, &a.DefaultPagePictureURL,
		&a.Enabled, &a.CreatedAt, &a.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.VaultTokenEncrypted = util.Decrypt(a.VaultTokenEncrypted)
	return &a, nil
}

func (r *MetaAccountRepo) List(ctx context.Context) ([]domain.MetaAccount, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+metaAccountCols+` FROM meta_accounts ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MetaAccount
	for rows.Next() {
		a, err := scanMetaAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *MetaAccountRepo) ListByAccount(ctx context.Context, accountID int64) ([]domain.MetaAccount, error) {
	if accountID == 0 {
		return r.List(ctx)
	}
	rows, err := r.pool.Query(ctx, `SELECT `+metaAccountCols+` FROM meta_accounts WHERE account_id = $1 ORDER BY id DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MetaAccount
	for rows.Next() {
		a, err := scanMetaAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (r *MetaAccountRepo) FindByID(ctx context.Context, id int64) (*domain.MetaAccount, error) {
	return scanMetaAccount(r.pool.QueryRow(ctx, `SELECT `+metaAccountCols+` FROM meta_accounts WHERE id = $1`, id))
}

func (r *MetaAccountRepo) FindFirstConnectedByAccount(ctx context.Context, accountID int64) (*domain.MetaAccount, error) {
	return scanMetaAccount(r.pool.QueryRow(ctx, `
		SELECT `+metaAccountCols+`
		FROM meta_accounts
		WHERE account_id = $1 AND enabled = TRUE AND vault_token_encrypted <> ''
		ORDER BY updated_at DESC, id DESC LIMIT 1
	`, accountID))
}

func (r *MetaAccountRepo) Create(ctx context.Context, a *domain.MetaAccount) error {
	if a.AccountID == 0 {
		a.AccountID = 1
	}
	encTok := util.Encrypt(a.VaultTokenEncrypted)
	return r.pool.QueryRow(ctx, `
		INSERT INTO meta_accounts (
			account_id, name, vault_token_encrypted,
			oauth_user_id, oauth_name, oauth_email, oauth_picture_url, token_expires_at,
			ad_account_id, ad_account_name, ad_account_picture_url,
			default_page_id, default_page_name, default_page_picture_url, enabled
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, created_at, updated_at
	`,
		a.AccountID, a.Name, encTok,
		a.OAuthUserID, a.OAuthName, a.OAuthEmail, a.OAuthPictureURL, a.TokenExpiresAt,
		a.AdAccountID, a.AdAccountName, a.AdAccountPictureURL,
		a.DefaultPageID, a.DefaultPageName, a.DefaultPagePictureURL, a.Enabled,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *MetaAccountRepo) Update(ctx context.Context, a *domain.MetaAccount) error {
	return r.pool.QueryRow(ctx, `
		UPDATE meta_accounts
		SET name = $1, ad_account_id = $2, ad_account_name = $3, ad_account_picture_url = $4,
		    default_page_id = $5, default_page_name = $6, default_page_picture_url = $7,
		    enabled = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING updated_at
	`,
		a.Name, a.AdAccountID, a.AdAccountName, a.AdAccountPictureURL,
		a.DefaultPageID, a.DefaultPageName, a.DefaultPagePictureURL,
		a.Enabled, a.ID,
	).Scan(&a.UpdatedAt)
}

func (r *MetaAccountRepo) SetOAuthTokens(ctx context.Context, id int64, vaultToken, userID, name, email, picURL string, expiresAt *time.Time) error {
	encTok := util.Encrypt(vaultToken)
	_, err := r.pool.Exec(ctx, `
		UPDATE meta_accounts
		SET vault_token_encrypted = $1, oauth_user_id = $2, oauth_name = $3,
		    oauth_email = $4, oauth_picture_url = $5, token_expires_at = $6, updated_at = NOW()
		WHERE id = $7
	`, encTok, userID, name, email, picURL, expiresAt, id)
	return err
}

func (r *MetaAccountRepo) UpdatePictures(ctx context.Context, id int64, oauthPic, adAccPic string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE meta_accounts
		SET oauth_picture_url = CASE WHEN $2 <> '' THEN $2 ELSE oauth_picture_url END,
		    ad_account_picture_url = CASE WHEN $3 <> '' THEN $3 ELSE ad_account_picture_url END
		WHERE id = $1
	`, id, oauthPic, adAccPic)
	return err
}

func (r *MetaAccountRepo) ClearOAuthTokens(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE meta_accounts
		SET vault_token_encrypted = '', oauth_user_id = '', oauth_name = '',
		    oauth_email = '', oauth_picture_url = '', ad_account_picture_url = '',
		    default_page_picture_url = '', token_expires_at = NULL, updated_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (r *MetaAccountRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM meta_accounts WHERE id = $1`, id)
	return err
}
