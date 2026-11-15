package ga4

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

type failingRoundTripper struct{}

func (failingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, &urlErrWrapper{url: req.URL.String()}
}

type urlErrWrapper struct {
	url string
}

func (e *urlErrWrapper) Error() string {
	return `Post "` + e.url + `": dial tcp: i/o timeout`
}

func TestMapEventName(t *testing.T) {
	cases := map[string]string{
		"QualifiedLead":                        "qualified_lead",
		"PixGerado":                            "pix_gerado",
		"Venda Aprovada!!":                     "venda_aprovada",
		"123Lead":                              "ev_123_lead",
		"evento___muito___longo_de_mais_de_40_caracteres_totais": "evento_muito_longo_de_mais_de_40_caracte",
		"   ":   "",
		"---!!": "",
	}
	for in, want := range cases {
		if got := mapEventName(in); got != want {
			t.Errorf("mapEventName(%q) = %q, want %q", in, got, want)
		}
		if got := mapEventName(in); len(got) > 40 {
			t.Errorf("mapEventName(%q) exceeded 40 chars: %d", in, len(got))
		}
	}
}

func TestSendEvents_TransportErrorPreservesRequestBodyAndRedactsSecret(t *testing.T) {
	c := &Client{httpClient: &http.Client{Transport: failingRoundTripper{}}}
	res := c.SendEvents(context.Background(), "SuperSecret999", "G-TEST123", &domain.SendEventInput{
		EventName:   "QualifiedLead",
		EventID:     "ev_1",
		ExternalID:  "ext_1",
		GA4ClientID: "111.222",
	})
	if res.Err == nil {
		t.Fatalf("expected error on transport failure")
	}
	if errors.Is(res.Err, ErrEventNotMapped) {
		t.Fatalf("QualifiedLead must be mapped")
	}
	if strings.Contains(res.Err.Error(), "SuperSecret999") {
		t.Fatalf("leaked api_secret in error: %v", res.Err)
	}
	if res.RequestBody == "" || !strings.Contains(res.RequestBody, `"qualified_lead"`) {
		t.Fatalf("expected RequestBody to be preserved on transport error, got %q", res.RequestBody)
	}
}
