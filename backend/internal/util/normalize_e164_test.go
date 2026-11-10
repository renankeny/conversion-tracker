package util

import (
	"sync"
	"testing"
)

func TestNormalizePhoneVsE164(t *testing.T) {
	cases := []struct {
		raw      string
		wantMeta string
		wantE164 string
	}{
		{"", "", ""},
		{"   ", "", ""},
		{"11999998888", "5511999998888", "+5511999998888"},
		{"(011) 98888-7777", "5511988887777", "+5511988887777"},
		{"+55 (21) 99876-5432", "5521998765432", "+5521998765432"},
		{"+1 (415) 555-2671", "14155552671", "+14155552671"},
	}

	for _, tc := range cases {
		gotMeta := NormalizePhone(tc.raw)
		if gotMeta != tc.wantMeta {
			t.Errorf("NormalizePhone(%q) = %q, want %q", tc.raw, gotMeta, tc.wantMeta)
		}
		gotE164 := NormalizePhoneE164(tc.raw)
		if gotE164 != tc.wantE164 {
			t.Errorf("NormalizePhoneE164(%q) = %q, want %q", tc.raw, gotE164, tc.wantE164)
		}
	}
}

func TestURLMapperAccountIsolationAndConcurrency(t *testing.T) {
	m := NewURLMapper()
	m.SetByAccount(map[int64]map[string]string{
		1: {"oferta.site.com": "https://institucional-a.com"},
		2: {"oferta.site.com": "https://institucional-b.com"},
	})

	gotA := m.ApplyForAccount(1, "https://oferta.site.com/pv?utm_source=fb")
	if gotA != "https://institucional-a.com/pv?utm_source=fb" {
		t.Fatalf("account 1 mapping got %q", gotA)
	}

	gotB := m.ApplyForAccount(2, "https://oferta.site.com/pv?utm_source=fb")
	if gotB != "https://institucional-b.com/pv?utm_source=fb" {
		t.Fatalf("account 2 mapping got %q", gotB)
	}

	// Conta 3 não possui regra para oferta.site.com -> não pode vazar regra da conta 1/2
	gotC := m.ApplyForAccount(3, "https://oferta.site.com/pv")
	if gotC != "https://oferta.site.com/pv" {
		t.Fatalf("account 3 leaked another account's mapping: %q", gotC)
	}

	// Garante ausência de deadlock/race com URLs relativas ou malformadas concorrentes a SetByAccount
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = m.ApplyForAccount(1, "/relative-only")
				_ = m.ApplyForAccount(1, "://bad-url")
				_ = m.ApplyForAccount(1, "https://oferta.site.com/checkout")
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 20; j++ {
			m.SetByAccount(map[int64]map[string]string{
				1: {"oferta.site.com": "https://institucional-a.com"},
			})
		}
	}()
	wg.Wait()
}

