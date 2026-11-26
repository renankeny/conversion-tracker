package util

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckSystemUpdateAndTriggerSelfUpdate(t *testing.T) {
	resetSystemUpdateCache()
	defer resetSystemUpdateCache()

	sc, err := newShardCipher()
	if err != nil {
		t.Fatalf("newShardCipher: %v", err)
	}

	var checkHits, triggerHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			Vector string `json:"v"`
		}
		if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var req updateCheckPayload
		if err := sc.openJSON(env.Vector, &req); err != nil || req.Identity != "owner@cliente.com" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch r.URL.Path {
		case "/api/v1/telemetry/update-check":
			checkHits++
			sealed, _ := sc.sealJSON(updateCheckWireResp{
				CurrentVersion:  req.AppVersion,
				LatestVersion:   "d8233c4",
				PublishedAt:     "2026-11-26T16:00:00Z",
				UpdateAvailable: true,
				CanAutoUpdate:   true,
				UpdateStatus:    "idle",
			})
			_ = json.NewEncoder(w).Encode(map[string]string{"v": sealed})
		case "/api/v1/telemetry/self-update":
			triggerHits++
			sealed, _ := sc.sealJSON(map[string]any{
				"ok":      true,
				"status":  "queued",
				"message": "Atualização iniciada!",
			})
			_ = json.NewEncoder(w).Encode(map[string]string{"v": sealed})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	prevVer := AppBuildVersion
	AppBuildVersion = "b44d14d"
	defer func() { AppBuildVersion = prevVer }()

	t.Setenv("APP_ENV", "development")
	t.Setenv("PGX_SHARD_SYNC_URI", srv.URL)

	cal := &ShardCalibrator{
		cipher:      sc,
		appDomain:   "app.cliente.com",
		instanceUID: "uid_test_12345678",
		client:      srv.Client(),
		cachedFrame: &ShardEpochFrame{Identity: "owner@cliente.com"},
	}

	st := cal.CheckSystemUpdate(context.Background(), false)
	if !st.UpdateAvailable || st.LatestVersion != "d8233c4" || !st.CanAutoUpdate {
		t.Fatalf("unexpected update status: %+v", st)
	}
	// Segunda chamada dentro do TTL deve reutilizar cache sem bater no servidor
	st2 := cal.CheckSystemUpdate(context.Background(), false)
	if checkHits != 1 || st2.LatestVersion != "d8233c4" {
		t.Fatalf("expected cached hit (checkHits=%d), got %+v", checkHits, st2)
	}

	status, msg, err := cal.TriggerSelfUpdate(context.Background())
	if err != nil || triggerHits != 1 || status != "queued" || msg == "" {
		t.Fatalf("unexpected TriggerSelfUpdate result: status=%q msg=%q err=%v hits=%d", status, msg, err, triggerHits)
	}
}

func TestCheckSystemUpdateFailOpenOnNotFound(t *testing.T) {
	resetSystemUpdateCache()
	defer resetSystemUpdateCache()

	sc, _ := newShardCipher()
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	t.Setenv("APP_ENV", "development")
	t.Setenv("PGX_SHARD_SYNC_URI", srv.URL)

	cal := &ShardCalibrator{
		cipher:      sc,
		client:      srv.Client(),
		cachedFrame: &ShardEpochFrame{Identity: "owner@cliente.com"},
	}
	st := cal.CheckSystemUpdate(context.Background(), true)
	if st.UpdateAvailable {
		t.Fatalf("expected fail-open update_available=false on 404, got %+v", st)
	}
}
