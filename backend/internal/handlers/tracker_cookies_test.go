package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetFirstPartyTrackingCookiesCNAME(t *testing.T) {
	reqPayload := sendRequest{
		UserID: "ABC1234567",
		FBP:    "fb.1.1700000000000.987654321",
		FBC:    "fb.1.1700000000000.IwAR12345",
		TTP:    "01JREALTTPCOOKIE_.tt.1",
	}

	// 1. CNAME 1st-party em domínio .com.br -> emite Set-Cookie com Domain=loja.com.br
	r := httptest.NewRequest(http.MethodPost, "https://track.loja.com.br/api/v1/events/collect", nil)
	r.Host = "track.loja.com.br"
	r.Header.Set("Origin", "https://www.loja.com.br")
	r.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	setFirstPartyTrackingCookies(rec, r, "api.trackeado.com.br", "track.loja.com.br", reqPayload)

	cookies := rec.Result().Cookies()
	if len(cookies) != 4 {
		t.Fatalf("expected 4 Set-Cookie headers on CNAME, got %d", len(cookies))
	}
	byName := make(map[string]*http.Cookie, len(cookies))
	for _, c := range cookies {
		byName[c.Name] = c
		if c.Domain != "loja.com.br" {
			t.Errorf("cookie %s domain = %q, want loja.com.br", c.Name, c.Domain)
		}
		if !c.Secure {
			t.Errorf("cookie %s expected Secure=true", c.Name)
		}
		if c.HttpOnly {
			t.Errorf("cookie %s expected HttpOnly=false for JS pixel sync", c.Name)
		}
	}
	if byName["tf_uid"] == nil || byName["tf_uid"].Value != "ABC1234567" {
		t.Errorf("missing or wrong tf_uid cookie")
	}
	if byName["_fbc"] == nil || byName["_fbc"].Value != reqPayload.FBC {
		t.Errorf("missing or wrong _fbc cookie")
	}

	// 2. Host compartilhado igual ao TRACKING_TARGET_HOST -> 0 cookies
	rShared := httptest.NewRequest(http.MethodPost, "https://api.trackeado.com.br/api/v1/events/collect", nil)
	rShared.Host = "api.trackeado.com.br"
	rShared.Header.Set("Origin", "https://www.loja.com.br")
	recShared := httptest.NewRecorder()

	setFirstPartyTrackingCookies(recShared, rShared, "api.trackeado.com.br", "", reqPayload)
	if len(recShared.Result().Cookies()) != 0 {
		t.Fatalf("expected 0 cookies on shared tracking target host, got %d", len(recShared.Result().Cookies()))
	}

	// 3. Origin de outro domínio tentando usar CNAME alheio -> 0 cookies
	rCross := httptest.NewRequest(http.MethodPost, "https://track.loja.com.br/api/v1/events/collect", nil)
	rCross.Host = "track.loja.com.br"
	rCross.Header.Set("Origin", "https://outrosite.com.br")
	recCross := httptest.NewRecorder()

	setFirstPartyTrackingCookies(recCross, rCross, "api.trackeado.com.br", "track.loja.com.br", reqPayload)
	if len(recCross.Result().Cookies()) != 0 {
		t.Fatalf("expected 0 cookies on cross-site origin, got %d", len(recCross.Result().Cookies()))
	}
}
