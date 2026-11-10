package domain

import (
	"strings"
	"time"
)

type MetaAccount struct {
	ID                    int64      `json:"id"`
	AccountID             int64      `json:"account_id"`
	Name                  string     `json:"name"`
	VaultTokenEncrypted   string     `json:"-"`
	OAuthUserID           string     `json:"oauth_user_id"`
	OAuthName             string     `json:"oauth_name"`
	OAuthEmail            string     `json:"oauth_email"`
	OAuthPictureURL       string     `json:"oauth_picture_url"`
	TokenExpiresAt        *time.Time `json:"token_expires_at,omitempty"`
	AdAccountID           string     `json:"ad_account_id"`
	AdAccountName         string     `json:"ad_account_name"`
	AdAccountPictureURL   string     `json:"ad_account_picture_url"`
	DefaultPageID         string     `json:"default_page_id"`
	DefaultPageName       string     `json:"default_page_name"`
	DefaultPagePictureURL string     `json:"default_page_picture_url"`
	Enabled               bool       `json:"enabled"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (a MetaAccount) IsConnected() bool {
	return strings.TrimSpace(a.VaultTokenEncrypted) != ""
}

func (a MetaAccount) HasAdAccount() bool {
	return a.IsConnected() && strings.TrimSpace(a.AdAccountID) != ""
}

func (a MetaAccount) EffectivePictureURL() string {
	if s := strings.TrimSpace(a.AdAccountPictureURL); s != "" {
		return s
	}
	return strings.TrimSpace(a.OAuthPictureURL)
}

func (a MetaAccount) PublicView() map[string]any {
	return map[string]any{
		"id":                       a.ID,
		"account_id":               a.AccountID,
		"name":                     a.Name,
		"connected":                a.IsConnected(),
		"oauth_connected":          a.IsConnected(),
		"oauth_user_id":            a.OAuthUserID,
		"oauth_name":               a.OAuthName,
		"oauth_email":              a.OAuthEmail,
		"oauth_picture_url":        a.OAuthPictureURL,
		"picture_url":              a.EffectivePictureURL(),
		"token_expires_at":         a.TokenExpiresAt,
		"ad_account_id":            a.AdAccountID,
		"ad_account_name":          a.AdAccountName,
		"ad_account_picture_url":   a.AdAccountPictureURL,
		"default_page_id":          a.DefaultPageID,
		"default_page_name":        a.DefaultPageName,
		"default_page_picture_url": a.DefaultPagePictureURL,
		"enabled":                  a.Enabled,
		"created_at":               a.CreatedAt,
		"updated_at":               a.UpdatedAt,
	}
}
