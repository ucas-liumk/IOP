package health

import (
	"context"
	"errors"
	"testing"
)

func TestRegistry_AllHealthy(t *testing.T) {
	r := NewRegistry()
	r.Register(Check{Name: "pg", Critical: true, Check: func(ctx context.Context) error { return nil }})
	r.Register(Check{Name: "redis", Critical: false, Check: func(ctx context.Context) error { return nil }})

	report := r.Report(context.Background())
	if !report.Ready {
		t.Fatalf("expected ready=true")
	}
	if !report.Live {
		t.Fatalf("expected live=true")
	}
}

func TestRegistry_CriticalDown(t *testing.T) {
	r := NewRegistry()
	r.Register(Check{Name: "pg", Critical: true, Check: func(ctx context.Context) error { return errors.New("down") }})
	report := r.Report(context.Background())
	if report.Ready {
		t.Fatalf("expected ready=false when critical down")
	}
	if !report.Live {
		t.Fatalf("expected live=true regardless")
	}
}

func TestRegistry_NoncriticalDown(t *testing.T) {
	r := NewRegistry()
	r.Register(Check{Name: "minio", Critical: false, Check: func(ctx context.Context) error { return errors.New("down") }})
	report := r.Report(context.Background())
	if !report.Ready {
		t.Fatalf("expected ready=true when noncritical down")
	}
}
