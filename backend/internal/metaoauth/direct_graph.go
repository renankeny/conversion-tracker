package metaoauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/meta"
)

var (
	metaDirectBaseURL   = "https://graph.facebook.com"
	metaDirectTransport http.RoundTripper
	pooledMetaTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   8 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   128,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
)

func newDirectMetaHTTPClient(timeout time.Duration) *http.Client {
	if metaDirectTransport != nil {
		return &http.Client{Timeout: timeout, Transport: metaDirectTransport}
	}
	return &http.Client{Timeout: timeout, Transport: pooledMetaTransport}
}

func directGraphGET(ctx context.Context, lease metaLeaseEntry, relPath string, params url.Values) ([]byte, int, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("access_token", lease.accessToken)
	params.Set("appsecret_proof", lease.appSecretProof)
	base := strings.TrimRight(metaDirectBaseURL, "/")
	u := fmt.Sprintf("%s/%s/%s?%s", base, lease.graphVersion, strings.TrimLeft(relPath, "/"), params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := newDirectMetaHTTPClient(15 * time.Second).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		status := resp.StatusCode
		var gErr struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &gErr)
		if gErr.Error.Code == 190 || strings.Contains(strings.ToLower(gErr.Error.Message), "validating access token") || strings.Contains(strings.ToLower(gErr.Error.Message), "invalid oauth access token") {
			status = http.StatusUnauthorized
		}
		msg := gErr.Error.Message
		if msg == "" {
			msg = fmt.Sprintf("Meta Graph API status %d", resp.StatusCode)
		}
		return raw, status, &ProxyError{StatusCode: status, Body: string(raw), Message: "Meta Graph API: " + msg}
	}
	return raw, resp.StatusCode, nil
}

func directSendCAPIEvent(ctx context.Context, lease metaLeaseEntry, pixelID string, payload []byte) (capiOutcome, int, error) {
	cleanPID := digitsOnlyStr(pixelID)
	if cleanPID == "" {
		return capiOutcome{}, http.StatusBadRequest, errors.New("pixel_id ausente")
	}
	q := url.Values{"appsecret_proof": {lease.appSecretProof}}
	u := fmt.Sprintf("%s/%s/%s/events?%s", strings.TrimRight(metaDirectBaseURL, "/"), lease.graphVersion, cleanPID, q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return capiOutcome{}, 0, errors.New("falha ao montar requisicao CAPI")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+lease.accessToken)
	resp, err := newDirectMetaHTTPClient(12 * time.Second).Do(req)
	if err != nil {
		msg := strings.ReplaceAll(err.Error(), lease.accessToken, "[REDACTED]")
		return capiOutcome{}, 0, fmt.Errorf("falha de rede Meta CAPI: %s", msg)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	bodyStr := string(raw)
	out := capiOutcome{StatusCode: resp.StatusCode, Body: bodyStr}
	if resp.StatusCode >= 300 {
		status := resp.StatusCode
		lower := strings.ToLower(bodyStr)
		if resp.StatusCode == http.StatusUnauthorized || strings.Contains(lower, "\"code\":190") || strings.Contains(lower, "validating access token") || strings.Contains(lower, "invalid oauth access token") {
			status = http.StatusUnauthorized
		}
		return out, status, &ProxyError{StatusCode: status, Body: bodyStr, Message: fmt.Sprintf("meta capi returned %d", resp.StatusCode)}
	}
	return out, resp.StatusCode, nil
}

func directGetAdMetadata(ctx context.Context, lease metaLeaseEntry, adID string) (AdMetadata, int, error) {
	cleanID := digitsOnlyStr(adID)
	if cleanID == "" {
		return AdMetadata{}, http.StatusBadRequest, errors.New("ad_id ausente")
	}
	q := url.Values{"fields": {"id,name,campaign{id,name},adset{id,name},creative{id,thumbnail_url,image_url},tracking_specs"}}
	raw, code, err := directGraphGET(ctx, lease, cleanID, q)
	if err != nil {
		return AdMetadata{}, code, err
	}
	var adData struct {
		Name          string                                  `json:"name"`
		Campaign      struct{ ID, Name string }               `json:"campaign"`
		Adset         struct{ ID, Name string }               `json:"adset"`
		Creative      struct{ ThumbnailURL, ImageURL string } `json:"creative"`
		TrackingSpecs []struct{ Page []string }               `json:"tracking_specs"`
	}
	if err := json.Unmarshal(raw, &adData); err != nil {
		return AdMetadata{}, http.StatusBadGateway, err
	}
	thumb := adData.Creative.ThumbnailURL
	if thumb == "" {
		thumb = adData.Creative.ImageURL
	}
	pageID := ""
	for _, sp := range adData.TrackingSpecs {
		if len(sp.Page) > 0 && sp.Page[0] != "" {
			pageID = sp.Page[0]
			break
		}
	}
	return AdMetadata{
		AdID: cleanID, AdName: adData.Name,
		CampaignID: adData.Campaign.ID, CampaignName: adData.Campaign.Name,
		AdsetID: adData.Adset.ID, AdsetName: adData.Adset.Name,
		PageID: pageID, ThumbnailURL: thumb,
	}, http.StatusOK, nil
}

func directFetchLiveInsights(ctx context.Context, lease metaLeaseEntry, adAccountID, timeRange string) (meta.LiveInsightsSummary, int, error) {
	act := normalizeMetaActID(adAccountID)
	if act == "" || act == "act_" {
		return meta.LiveInsightsSummary{}, http.StatusBadRequest, errors.New("ad_account_id ausente")
	}
	q := url.Values{"level": {"account"}, "fields": {"spend,clicks,impressions,actions,action_values"}, "time_range": {timeRange}}
	raw, code, err := directGraphGET(ctx, lease, act+"/insights", q)
	if err != nil {
		return meta.LiveInsightsSummary{}, code, err
	}
	return parseDirectLiveInsights(raw), http.StatusOK, nil
}
