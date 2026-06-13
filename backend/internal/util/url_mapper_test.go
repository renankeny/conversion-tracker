package util

import "testing"

func TestURLMapper_Apply_NoMatch(t *testing.T) {
	m := NewURLMapper()
	m.Set(map[string]string{"black1.com": "white1.com"})
	in := "https://outro.com/produto?utm_source=fb"
	if got := m.Apply(in); got != in {
		t.Errorf("expected unchanged, got %s", got)
	}
}

func TestURLMapper_Apply_HostOnly(t *testing.T) {
	m := NewURLMapper()
	m.Set(map[string]string{"black1.com": "https://white1.com"})
	got := m.Apply("https://black1.com/landing?utm_source=fb")
	want := "https://white1.com/landing?utm_source=fb"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestURLMapper_Apply_HostAndPathExact(t *testing.T) {
	m := NewURLMapper()
	m.Set(map[string]string{
		"https://black1.com/produto": "https://white1.com/p",
		"black1.com":                  "https://other.com",
	})
	// match mais específico (com scheme+path) deve ganhar
	got := m.Apply("https://black1.com/produto?fbclid=abc")
	want := "https://white1.com/p?fbclid=abc"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestURLMapper_Apply_PreservesFragment(t *testing.T) {
	m := NewURLMapper()
	m.Set(map[string]string{"black.com": "https://white.com"})
	got := m.Apply("https://black.com/page#section")
	if got != "https://white.com/page#section" {
		t.Errorf("got %s", got)
	}
}

func TestURLMapper_Apply_EmptyMapper(t *testing.T) {
	m := NewURLMapper()
	in := "https://x.com/y"
	if got := m.Apply(in); got != in {
		t.Errorf("got %s", got)
	}
}
