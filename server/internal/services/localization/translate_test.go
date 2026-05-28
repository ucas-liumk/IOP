package localization

import (
	"context"
	"testing"
)

func TestT_KnownKey(t *testing.T) {
	svc := NewService(MapBundle(map[string]map[string]string{
		"zh-CN": {"okr.plan.invalid_period": "时段不合法"},
	}), "zh-CN")
	if got := svc.T(context.Background(), "okr.plan.invalid_period"); got != "时段不合法" {
		t.Fatalf("got %q", got)
	}
}

func TestT_UnknownKey_ReturnsKey(t *testing.T) {
	svc := NewService(MapBundle(nil), "zh-CN")
	if got := svc.T(context.Background(), "missing.key"); got != "missing.key" {
		t.Fatalf("expected key fallback, got %q", got)
	}
}

func TestT_TemplateArgs(t *testing.T) {
	svc := NewService(MapBundle(map[string]map[string]string{
		"zh-CN": {"hello": "你好, {name}"},
	}), "zh-CN")
	got := svc.T(context.Background(), "hello", "name", "leo")
	if got != "你好, leo" {
		t.Fatalf("got %q", got)
	}
}
