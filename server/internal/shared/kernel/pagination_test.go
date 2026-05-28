package kernel

import "testing"

func TestPagination_Defaults(t *testing.T) {
	p := Pagination{}.Normalize()
	if p.Page != 1 || p.PageSize != 20 {
		t.Fatalf("expected defaults page=1 pageSize=20, got %+v", p)
	}
}

func TestPagination_Clamp(t *testing.T) {
	p := Pagination{Page: -5, PageSize: 9999}.Normalize()
	if p.Page != 1 {
		t.Fatalf("expected page clamped to 1, got %d", p.Page)
	}
	if p.PageSize != 200 {
		t.Fatalf("expected pageSize clamped to max 200, got %d", p.PageSize)
	}
}

func TestPagination_Offset(t *testing.T) {
	p := Pagination{Page: 3, PageSize: 50}
	if p.Offset() != 100 {
		t.Fatalf("expected offset 100, got %d", p.Offset())
	}
}
