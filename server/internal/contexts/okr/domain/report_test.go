package domain

import (
	"testing"
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

func TestNewDailyReport_RequiresOneDay(t *testing.T) {
	owner := kernel.NewID()
	day := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	period := Period{Start: day, End: day}
	r, err := NewDailyReport(owner, period, "did things", day)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Type != ReportDaily {
		t.Fatalf("type mismatch")
	}

	twoDay := Period{Start: day, End: day.Add(24 * time.Hour)}
	if _, err := NewDailyReport(owner, twoDay, "x", day); err == nil {
		t.Fatalf("expected wrong-period err")
	}
}

func TestNewWeeklyReport_RequiresMondaySunday(t *testing.T) {
	owner := kernel.NewID()
	monday := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC) // 2026-02-02 is Monday
	week := Period{Start: monday, End: monday.AddDate(0, 0, 6)}
	if _, err := NewWeeklyReport(owner, week, "x", monday); err != nil {
		t.Fatalf("week monday-sun should be ok: %v", err)
	}
	// Tuesday-Monday
	bad := Period{Start: monday.AddDate(0, 0, 1), End: monday.AddDate(0, 0, 7)}
	if _, err := NewWeeklyReport(owner, bad, "x", monday); err == nil {
		t.Fatalf("expected weekly period err")
	}
}

func TestReport_AddEntry(t *testing.T) {
	day := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	r, _ := NewDailyReport(kernel.NewID(), Period{Start: day, End: day}, "", day)
	r.AddEntry("ship doc", "detail", "70%", nil)
	r.AddEntry("review pr", "", "", nil)
	if len(r.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(r.Entries))
	}
	if r.Entries[1].SortOrder != 2 {
		t.Fatalf("sort order wrong")
	}
}
