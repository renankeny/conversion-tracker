package util

import (
	"testing"
	"time"
)

func TestClampEventTime(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	if got := ClampEventTime(now.Unix()*1000, now); got != now.Unix() {
		t.Fatalf("ms not converted: %d", got)
	}
	if got := ClampEventTime(now.Add(time.Hour).Unix(), now); got != now.Unix() {
		t.Fatalf("future not clamped: %d", got)
	}
	if got := ClampEventTime(now.Add(-30*24*time.Hour).Unix(), now); got < now.Add(-7*24*time.Hour).Unix() {
		t.Fatalf("old not clamped: %d", got)
	}
	ok := now.Add(-time.Hour).Unix()
	if got := ClampEventTime(ok, now); got != ok {
		t.Fatalf("valid changed: %d", got)
	}
}
