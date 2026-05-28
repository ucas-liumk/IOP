package kernel

import (
	"testing"
	"time"
)

func TestRealClock(t *testing.T) {
	c := RealClock{}
	now := c.Now()
	if now.IsZero() {
		t.Fatalf("expected non-zero time")
	}
	if time.Since(now) > time.Second {
		t.Fatalf("clock drift too large")
	}
}

func TestFakeClock(t *testing.T) {
	fixed := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	c := NewFakeClock(fixed)
	if !c.Now().Equal(fixed) {
		t.Fatalf("expected %v, got %v", fixed, c.Now())
	}
	c.Advance(2 * time.Hour)
	expected := fixed.Add(2 * time.Hour)
	if !c.Now().Equal(expected) {
		t.Fatalf("expected %v after advance, got %v", expected, c.Now())
	}
}
