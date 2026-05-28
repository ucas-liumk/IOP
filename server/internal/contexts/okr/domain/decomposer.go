package domain

// PlanDecomposer enforces parent→child cross-aggregate invariants.
// (We don't model "parent contains children" as a single aggregate to keep them small.)
type PlanDecomposer struct{}

// Validate checks that childLevel is narrower than parent.Level and that
// child period is fully contained inside parent period.
func (PlanDecomposer) Validate(parent *Plan, childLevel PlanLevel, childPeriod Period) error {
	if !parent.Level.IsBroaderThan(childLevel) {
		return ErrChildLevel
	}
	if !(childPeriod.Start.After(parent.Period.Start) || childPeriod.Start.Equal(parent.Period.Start)) {
		return ErrChildOutside
	}
	if !(childPeriod.End.Before(parent.Period.End) || childPeriod.End.Equal(parent.Period.End)) {
		return ErrChildOutside
	}
	return nil
}
