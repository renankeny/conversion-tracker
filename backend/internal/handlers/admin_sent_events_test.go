package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestChiURLParamEncoding(t *testing.T) {
	r := chi.NewRouter()
	var captured string
	r.Get("/test/{event_id}", func(w http.ResponseWriter, r *http.Request) {
		raw := chi.URLParam(r, "event_id")
		if unescaped, err := url.PathUnescape(raw); err == nil && unescaped != "" {
			captured = unescaped
		} else {
			captured = raw
		}
		w.WriteHeader(http.StatusOK)
	})

	rawURL := "/test/ctwa%3AAfgXdCAaFOpZsTzywANCp7ri0t99XU"
	req := httptest.NewRequest("GET", rawURL, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expected := "ctwa:AfgXdCAaFOpZsTzywANCp7ri0t99XU"
	if captured != expected {
		t.Fatalf("expected unescaped %q, got %q", expected, captured)
	}
}
