package kernel

import (
	"testing"
	"time"
)

func TestNewID_Unique(t *testing.T) {
	a := NewID()
	b := NewID()
	if a == b {
		t.Fatalf("expected different IDs, got %s and %s", a, b)
	}
}

func TestNewID_TimeOrdered(t *testing.T) {
	a := NewID()
	time.Sleep(2 * time.Millisecond)
	b := NewID()
	if a >= b {
		t.Fatalf("expected UUID v7 to be time-ordered, got a=%s >= b=%s", a, b)
	}
}

func TestParseID_Valid(t *testing.T) {
	id := NewID()
	parsed, err := ParseID(string(id))
	if err != nil {
		t.Fatalf("expected valid parse, got %v", err)
	}
	if parsed != id {
		t.Fatalf("round-trip mismatch")
	}
}

func TestParseID_Invalid(t *testing.T) {
	if _, err := ParseID("not-a-uuid"); err == nil {
		t.Fatalf("expected error on invalid uuid")
	}
}
