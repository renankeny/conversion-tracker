package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func (h *AdminMetaAccounts) maybeBackfillPicture(ctx context.Context, a *domain.MetaAccount) {
	if !a.IsConnected() || a.EffectivePictureURL() != "" || h.client == nil {
		return
	}
	checkKey := fmt.Sprintf("tf:meta:pic_checked:%d", a.ID)
	if h.rdb != nil {
		if exists, _ := h.rdb.Exists(ctx, checkKey).Result(); exists > 0 {
			return
		}
		_ = h.rdb.Set(ctx, checkKey, "1", 6*time.Hour).Err()
	}
	res, err := h.client.ListAdAccounts(ctx, a)
	if err != nil {
		return
	}
	oauthPic, _ := res["oauth_picture_url"].(string)
	adPic := extractMatchedAdAccountPic(res["ad_accounts"], a.AdAccountID)
	if oauthPic != "" || adPic != "" {
		_ = h.accounts.UpdatePictures(ctx, a.ID, oauthPic, adPic)
	}
}

func extractMatchedAdAccountPic(rawItems any, targetAdAccID string) string {
	items, ok := rawItems.([]any)
	if !ok {
		return ""
	}
	for _, raw := range items {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		idStr, _ := m["id"].(string)
		accStr, _ := m["account_id"].(string)
		if targetAdAccID != "" && (idStr == targetAdAccID || "act_"+accStr == targetAdAccID) {
			pic, _ := m["picture_url"].(string)
			return pic
		}
	}
	if len(items) == 1 {
		if m, ok := items[0].(map[string]any); ok {
			pic, _ := m["picture_url"].(string)
			return pic
		}
	}
	return ""
}

func (h *AdminMetaAccounts) backfillPicturesAsync(list []domain.MetaAccount) {
	var pending []domain.MetaAccount
	for _, item := range list {
		if item.IsConnected() && item.EffectivePictureURL() == "" {
			pending = append(pending, item)
		}
	}
	if len(pending) == 0 {
		return
	}
	go func(items []domain.MetaAccount) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		for i := range items {
			h.maybeBackfillPicture(ctx, &items[i])
		}
	}(pending)
}

func (h *AdminMetaAccounts) cachedRemoteMap(ctx context.Context, cacheKey string, bypass bool, loader func() (map[string]any, error)) (map[string]any, error) {
	if !bypass && h.rdb != nil && cacheKey != "" {
		if raw, err := h.rdb.Get(ctx, cacheKey).Bytes(); err == nil && len(raw) > 0 {
			var cached map[string]any
			if json.Unmarshal(raw, &cached) == nil {
				return cached, nil
			}
		}
	}
	res, err := loader()
	if err != nil {
		return nil, err
	}
	if h.rdb != nil && cacheKey != "" && res != nil {
		if raw, merr := json.Marshal(res); merr == nil {
			_ = h.rdb.Set(ctx, cacheKey, raw, 3*time.Minute).Err()
		}
	}
	return res, nil
}

func (h *AdminMetaAccounts) ListAdAccounts(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	refresh := r.URL.Query().Get("refresh") == "1"
	key := fmt.Sprintf("tf:meta:adaccs:%d:%d", a.ID, a.UpdatedAt.UnixNano())
	res, err := h.cachedRemoteMap(r.Context(), key, refresh, func() (map[string]any, error) {
		return h.client.ListAdAccounts(r.Context(), a)
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if oauthPic, _ := res["oauth_picture_url"].(string); oauthPic != "" && a.OAuthPictureURL == "" {
		_ = h.accounts.UpdatePictures(r.Context(), a.ID, oauthPic, "")
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AdminMetaAccounts) ListPages(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	refresh := r.URL.Query().Get("refresh") == "1"
	key := fmt.Sprintf("tf:meta:pages:%d:%d", a.ID, a.UpdatedAt.UnixNano())
	res, err := h.cachedRemoteMap(r.Context(), key, refresh, func() (map[string]any, error) {
		return h.client.ListPages(r.Context(), a)
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AdminMetaAccounts) ListPixels(w http.ResponseWriter, r *http.Request) {
	a, ok := h.loadAuthorizedAccount(w, r)
	if !ok {
		return
	}
	adAccID := strings.TrimSpace(r.URL.Query().Get("ad_account_id"))
	refresh := r.URL.Query().Get("refresh") == "1"
	key := fmt.Sprintf("tf:meta:pixels:%d:%d:%s", a.ID, a.UpdatedAt.UnixNano(), adAccID)
	res, err := h.cachedRemoteMap(r.Context(), key, refresh, func() (map[string]any, error) {
		return h.client.ListPixels(r.Context(), a, adAccID)
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}
