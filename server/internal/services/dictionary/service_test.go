package dictionary

import (
	"context"
	"testing"
)

func TestLookup_ReturnsSeededItems(t *testing.T) {
	svc := NewService(MemoryRepo(map[string][]Item{
		"plan_level": {
			{Code: "year", Name: "年度", SortOrder: 1, Active: true},
			{Code: "month", Name: "月度", SortOrder: 2, Active: true},
		},
	}))
	items, err := svc.Lookup(context.Background(), "plan_level")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestLookup_UnknownType(t *testing.T) {
	svc := NewService(MemoryRepo(map[string][]Item{}))
	_, err := svc.Lookup(context.Background(), "nope")
	if err == nil {
		t.Fatalf("expected error for unknown type")
	}
}

func TestLookup_FiltersInactive(t *testing.T) {
	svc := NewService(MemoryRepo(map[string][]Item{
		"x": {
			{Code: "a", Active: true},
			{Code: "b", Active: false},
		},
	}))
	items, _ := svc.Lookup(context.Background(), "x")
	if len(items) != 1 {
		t.Fatalf("expected only active items, got %d", len(items))
	}
}
