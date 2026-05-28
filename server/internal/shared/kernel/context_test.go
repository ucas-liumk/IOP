package kernel

import (
	"context"
	"testing"
)

func TestContext_TraceID(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-abc")
	if got := TraceIDFromContext(ctx); got != "trace-abc" {
		t.Fatalf("expected trace-abc, got %q", got)
	}
}

func TestContext_TraceID_Missing(t *testing.T) {
	if got := TraceIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestContext_TenantID(t *testing.T) {
	tid := NewID()
	ctx := WithTenantID(context.Background(), tid)
	got, ok := TenantIDFromContext(ctx)
	if !ok || got != tid {
		t.Fatalf("expected %s, got %s ok=%v", tid, got, ok)
	}
}

func TestContext_MemberID(t *testing.T) {
	mid := NewID()
	ctx := WithMemberID(context.Background(), mid)
	got, ok := MemberIDFromContext(ctx)
	if !ok || got != mid {
		t.Fatalf("expected %s, got %s ok=%v", mid, got, ok)
	}
}
