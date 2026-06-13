package util

import "testing"

func TestDomainSet_AllowOrigin_Exact(t *testing.T) {
	s := NewDomainSet()
	s.Set([]string{"foo.com", "bar.com"})

	if !s.AllowOrigin("https://foo.com") {
		t.Error("expected foo.com allowed")
	}
	if !s.AllowOrigin("https://bar.com:443") {
		t.Error("expected bar.com:443 allowed (Hostname strips port)")
	}
	if s.AllowOrigin("https://baz.com") {
		t.Error("baz.com should not be allowed")
	}
}

func TestDomainSet_Wildcard(t *testing.T) {
	s := NewDomainSet()
	s.Set([]string{"*.example.com"})

	if !s.AllowHost("api.example.com") {
		t.Error("api.example.com should match *.example.com")
	}
	if !s.AllowHost("a.b.example.com") {
		t.Error("a.b.example.com should match *.example.com")
	}
	if !s.AllowHost("example.com") {
		t.Error("apex example.com should match *.example.com")
	}
	if s.AllowHost("badexample.com") {
		t.Error("badexample.com must NOT match *.example.com")
	}
}

func TestDomainSet_EmptyEntries(t *testing.T) {
	s := NewDomainSet()
	s.Set([]string{"", "  ", "a.com"})

	if s.AllowHost("") {
		t.Error("empty host must not be allowed")
	}
	if !s.AllowHost("a.com") {
		t.Error("a.com should be allowed")
	}
	if s.Size() != 1 {
		t.Errorf("expected size 1, got %d", s.Size())
	}
}
