package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func extractOrSynthesizeBridgeIDs(r *http.Request) (userID, idWpp, fbp, fbc, ttp, ga4CID string) {
	nowMs := time.Now().UnixMilli()
	nowSec := nowMs / 1000

	if c, err := r.Cookie("__userData"); err == nil && c.Value != "" {
		if decoded, derr := url.QueryUnescape(c.Value); derr == nil {
			var ud struct {
				UserID string `json:"userId"`
				IDWpp  string `json:"id_wpp"`
			}
			if json.Unmarshal([]byte(decoded), &ud) == nil {
				if validCookieValue(ud.UserID, 64) {
					userID = ud.UserID
				}
				if validCookieValue(ud.IDWpp, 64) {
					idWpp = ud.IDWpp
				}
			}
		}
	}
	if userID == "" {
		if c, err := r.Cookie("tf_uid"); err == nil && validCookieValue(c.Value, 64) {
			userID = c.Value
		} else {
			userID = util.GenerateID10()
		}
	}
	if idWpp == "" {
		idWpp = userID
	}

	if c, err := r.Cookie("_fbp"); err == nil && strings.HasPrefix(c.Value, "fb.") {
		fbp = c.Value
	} else {
		fbp = fmt.Sprintf("fb.1.%d.%d", nowMs, 1000000000+rand.Int64N(8999999999))
	}
	if fbclid := strings.TrimSpace(r.URL.Query().Get("fbclid")); fbclid != "" {
		fbc = fmt.Sprintf("fb.1.%d.%s", nowMs, fbclid)
	} else if c, err := r.Cookie("_fbc"); err == nil && strings.HasPrefix(c.Value, "fb.") {
		fbc = c.Value
	}
	if c, err := r.Cookie("_ttp"); err == nil && validCookieValue(c.Value, 128) {
		ttp = c.Value
	}
	if c, err := r.Cookie("_ga"); err == nil && strings.HasPrefix(c.Value, "GA") {
		parts := strings.Split(c.Value, ".")
		if len(parts) >= 4 {
			ga4CID = parts[len(parts)-2] + "." + parts[len(parts)-1]
		}
	}
	if ga4CID == "" {
		ga4CID = strconv.FormatInt(100000000+rand.Int64N(899999999), 10) + "." + strconv.FormatInt(nowSec, 10)
	}
	return userID, idWpp, fbp, fbc, ttp, ga4CID
}

func setBridgeTrackingCookies(w http.ResponseWriter, r *http.Request, trackingTargetHost, pixelTrackingHost string, req sendRequest) {
	pxHost := stripHostPort(pixelTrackingHost)
	reqHost := stripHostPort(effectiveRequestHost(r))
	if pxHost != "" && strings.EqualFold(reqHost, pxHost) {
		setFirstPartyTrackingCookies(w, r, trackingTargetHost, pixelTrackingHost, req)
		return
	}
	secure := isHTTPSRequest(r)
	if validCookieValue(req.UserID, 64) {
		http.SetCookie(w, &http.Cookie{Name: "tf_uid", Value: req.UserID, Path: "/w", MaxAge: 365 * 86400, SameSite: http.SameSiteLaxMode, Secure: secure})
	}
	if strings.HasPrefix(req.FBP, "fb.") && validCookieValue(req.FBP, 128) {
		http.SetCookie(w, &http.Cookie{Name: "_fbp", Value: req.FBP, Path: "/w", MaxAge: 90 * 86400, SameSite: http.SameSiteLaxMode, Secure: secure})
	}
	if strings.HasPrefix(req.FBC, "fb.") && validCookieValue(req.FBC, 256) {
		http.SetCookie(w, &http.Cookie{Name: "_fbc", Value: req.FBC, Path: "/w", MaxAge: 90 * 86400, SameSite: http.SameSiteLaxMode, Secure: secure})
	}
}
