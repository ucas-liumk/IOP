package domain

import (
	"testing"
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

var (
	someTime = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

func TestNewPlan_Valid(t *testing.T) {
	period := Period{Start: someTime, End: someTime.Add(7 * 24 * time.Hour)}
	p, err := NewPlan(LevelWeek, kernel.NewID(), period, "学 DDD", nil, someTime)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.Status != PlanDraft {
		t.Fatalf("expected draft, got %s", p.Status)
	}
}

func TestNewPlan_InvalidLevel(t *testing.T) {
	period := Period{Start: someTime, End: someTime.Add(time.Hour)}
	if _, err := NewPlan("bogus", kernel.NewID(), period, "x", nil, someTime); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewPlan_InvalidPeriod(t *testing.T) {
	period := Period{Start: someTime, End: someTime} // zero length
	if _, err := NewPlan(LevelWeek, kernel.NewID(), period, "x", nil, someTime); err == nil {
		t.Fatal("expected error")
	}
}

func TestPlan_AddItem_WeightSumInvariant(t *testing.T) {
	p, _ := NewPlan(LevelWeek, kernel.NewID(),
		Period{Start: someTime, End: someTime.Add(7 * 24 * time.Hour)}, "x", nil, someTime)
	if _, err := p.AddItem("a", 60, someTime); err != nil {
		t.Fatalf("a: %v", err)
	}
	if _, err := p.AddItem("b", 40, someTime); err != nil {
		t.Fatalf("b: %v", err)
	}
	if _, err := p.AddItem("c", 1, someTime); err != ErrWeightOverflow {
		t.Fatalf("expected weight overflow, got %v", err)
	}
}

func TestPlan_CompleteItem_ProgressBecomes100(t *testing.T) {
	p, _ := NewPlan(LevelWeek, kernel.NewID(),
		Period{Start: someTime, End: someTime.Add(7 * 24 * time.Hour)}, "x", nil, someTime)
	it, _ := p.AddItem("a", 50, someTime)
	if err := p.CompleteItem(it.ID, "done", someTime); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if it.ProgressPct != 100 || it.Status != ItemDone {
		t.Fatalf("expected 100/done, got %d/%s", it.ProgressPct, it.Status)
	}
}

func TestPlan_ClosedPlanRejectsEdits(t *testing.T) {
	p, _ := NewPlan(LevelWeek, kernel.NewID(),
		Period{Start: someTime, End: someTime.Add(7 * 24 * time.Hour)}, "x", nil, someTime)
	_ = p.Close(someTime)
	if _, err := p.AddItem("a", 1, someTime); err != ErrPlanClosed {
		t.Fatalf("expected closed err, got %v", err)
	}
}

func TestPlan_OverallProgress_WeightedAvg(t *testing.T) {
	p, _ := NewPlan(LevelWeek, kernel.NewID(),
		Period{Start: someTime, End: someTime.Add(7 * 24 * time.Hour)}, "x", nil, someTime)
	a, _ := p.AddItem("a", 30, someTime)
	b, _ := p.AddItem("b", 70, someTime)
	_ = p.UpdateItemProgress(a.ID, 100, ItemDone, "", someTime)
	_ = p.UpdateItemProgress(b.ID, 50, ItemDoing, "", someTime)
	got := p.OverallProgress() // 30*100 + 70*50 = 6500 / 100 = 65
	if got != 65 {
		t.Fatalf("expected 65, got %d", got)
	}
}

func TestDecomposer_RejectsWideningLevel(t *testing.T) {
	parent, _ := NewPlan(LevelMonth, kernel.NewID(),
		Period{Start: someTime, End: someTime.AddDate(0, 1, 0)}, "month", nil, someTime)
	childPeriod := Period{Start: someTime, End: someTime.AddDate(0, 1, 0)}
	if err := (PlanDecomposer{}).Validate(parent, LevelYear, childPeriod); err != ErrChildLevel {
		t.Fatalf("expected level err, got %v", err)
	}
}

func TestDecomposer_RejectsPeriodOutsideParent(t *testing.T) {
	parent, _ := NewPlan(LevelMonth, kernel.NewID(),
		Period{Start: someTime, End: someTime.AddDate(0, 0, 28)}, "month", nil, someTime)
	out := Period{Start: someTime.AddDate(0, 0, 25), End: someTime.AddDate(0, 0, 32)}
	if err := (PlanDecomposer{}).Validate(parent, LevelWeek, out); err != ErrChildOutside {
		t.Fatalf("expected outside err, got %v", err)
	}
}

func TestCadence_WeekStartsMonday(t *testing.T) {
	// 2026-01-07 is a Wednesday
	ref := time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC)
	period := Cadence{}.Compute(LevelWeek, ref)
	if period.Start.Weekday() != time.Monday {
		t.Fatalf("expected monday, got %s", period.Start.Weekday())
	}
	if period.Days() != 7 {
		t.Fatalf("expected 7 days, got %d", period.Days())
	}
}

func TestCadence_MonthBounds(t *testing.T) {
	ref := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	period := Cadence{}.Compute(LevelMonth, ref)
	if period.Start.Day() != 1 || period.End.Day() != 28 {
		t.Fatalf("expected feb-2026 1..28, got %v..%v", period.Start, period.End)
	}
}
