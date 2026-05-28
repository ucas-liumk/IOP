package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	e := New(KindBusiness, "okr.plan.invalid_period", "period must be non-empty")
	if e.Kind != KindBusiness {
		t.Fatalf("kind mismatch")
	}
	if e.Code != "okr.plan.invalid_period" {
		t.Fatalf("code mismatch")
	}
	if e.Error() != "okr.plan.invalid_period: period must be non-empty" {
		t.Fatalf("error string mismatch: %q", e.Error())
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("connection refused")
	e := Wrap(KindDatabase, "db.connection_failed", "cannot connect to PG", cause)
	if !errors.Is(e, cause) {
		t.Fatalf("expected errors.Is to find cause")
	}
}

func TestAs(t *testing.T) {
	e := New(KindAuth, "iam.invalid_password", "wrong password")
	var target *Error
	if !errors.As(e, &target) {
		t.Fatalf("expected errors.As to match")
	}
	if target.Code != "iam.invalid_password" {
		t.Fatalf("As did not set target")
	}
}

func TestKind_String(t *testing.T) {
	if KindBusiness.String() != "business" {
		t.Fatalf("kind name mismatch")
	}
	if KindUnknown.String() != "unknown" {
		t.Fatalf("kind name mismatch")
	}
}
