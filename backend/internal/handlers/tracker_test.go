package handlers

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTrackerTemplate_RendersValidJSAndFixes(t *testing.T) {
	th, err := NewTracker(nil, "http://localhost:8080/api/v1")
	if err != nil {
		t.Fatalf("NewTracker failed: %v", err)
	}

	var loaderBuf bytes.Buffer
	err = th.loaderTpl.Execute(&loaderBuf, map[string]any{
		"PixelName":   "teste",
		"APIURL":      "http://localhost:8080/api/v1",
		"GeneratedAt": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("failed to execute loader template: %v", err)
	}

	var buf bytes.Buffer
	data := map[string]any{
		"PixelName":             "teste",
		"PixelNameJSON":         `"teste"`,
		"ConfigJSON":            `{"product_checkout":"pay","auto_checkout_buttons":true,"auto_page_view":true,"product_currency":"BRL","product_value":[99.9],"utm_origin_param":"utm_source","page_rules":[{"page":"/obrigado","event":"Lead"}]}`,
		"MetaInitPixelIDJSON":   `"123456"`,
		"MetaPixelIDsJSON":      `["123456","789012"]`,
		"TiktokInitPixelIDJSON": `"TT111"`,
		"TiktokPixelIDsJSON":    `["TT111","TT222"]`,
		"GA4MeasurementIDJSON":  `"G-TEST123"`,
		"APIURLJSON":            `"http://localhost:8080/api/v1"`,
		"APIURL":                "http://localhost:8080/api/v1",
		"GeneratedAt":           time.Now().UTC().Format(time.RFC3339),
	}

	if err := th.trackerTpl.Execute(&buf, data); err != nil {
		t.Fatalf("failed to execute tracker template: %v", err)
	}

	rendered := buf.String()

	for _, mustHave := range []string{
		"matchesCheckout",
		"getEffectiveUTMs",
		"stripUserMarker",
		"resolveButtonPrice",
		"observeDOM",
		"link.href = newHref",
		`"123456","789012"`,
		`"TT111","TT222"`,
	} {
		if !strings.Contains(rendered, mustHave) {
			t.Errorf("rendered tracker missing %q", mustHave)
		}
	}

	if nodePath, err := exec.LookPath("node"); err == nil {
		tmpDir := t.TempDir()
		loaderFile := filepath.Join(tmpDir, "loader.js")
		trackerFile := filepath.Join(tmpDir, "tracker.js")
		_ = os.WriteFile(loaderFile, loaderBuf.Bytes(), 0600)
		_ = os.WriteFile(trackerFile, buf.Bytes(), 0600)

		if out, err := exec.Command(nodePath, "--check", loaderFile).CombinedOutput(); err != nil {
			t.Fatalf("loader.js syntax error: %v\n%s", err, string(out))
		}
		if out, err := exec.Command(nodePath, "--check", trackerFile).CombinedOutput(); err != nil {
			t.Fatalf("tracker.js syntax error: %v\n%s", err, string(out))
		}
	}
}
