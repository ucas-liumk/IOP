package tenancy

import (
	"testing"

	"github.com/leo/iop/server/internal/shared/kernel"
)

func TestDeptImportParserReportsExcelRowNumber(t *testing.T) {
	res := kernel.NewBulkResult()
	rows := parseDeptImportRows([][]string{
		deptImportHeader,
		{"技术部", "", "ROOT", "department", "", "", "1", "active", ""},
		{"后端组", "TECH-BE", "TECH", "team", "", "", "2", "active", ""},
	}, res)
	if len(rows) != 1 {
		t.Fatalf("expected 1 valid row, got %d", len(rows))
	}
	if res.Failed != 1 || len(res.Errors) != 1 {
		t.Fatalf("expected one validation error, got failed=%d errors=%v", res.Failed, res.Errors)
	}
	if res.Errors[0].Row != 2 {
		t.Fatalf("expected Excel row 2, got %d", res.Errors[0].Row)
	}
}

func TestDeptImportCycleValidation(t *testing.T) {
	res := kernel.NewBulkResult()
	rows := parseDeptImportRows([][]string{
		deptImportHeader,
		{"技术部", "TECH", "TECH-BE", "department", "", "", "1", "active", ""},
		{"后端组", "TECH-BE", "TECH", "team", "", "", "2", "active", ""},
	}, res)
	byCode := map[string]deptImportRow{}
	for _, r := range rows {
		byCode[lowerCode(r.orgCode)] = r
	}
	failImportCycles(rows, byCode, res)
	if res.Failed == 0 {
		t.Fatal("expected cycle validation failure")
	}
	if res.Errors[0].Row != 2 {
		t.Fatalf("expected first cycle error on row 2, got %d", res.Errors[0].Row)
	}
}
