package domain

import "time"

// Cadence is a domain service that converts (level, date) into the canonical Period.
// Used to drive both Plan creation defaults and Report period validation.
type Cadence struct{}

func (Cadence) Compute(level PlanLevel, ref time.Time) Period {
	ref = ref.UTC()
	switch level {
	case LevelYear:
		start := time.Date(ref.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(ref.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
		return Period{Start: start, End: end}
	case LevelHalfYear:
		if ref.Month() <= 6 {
			return Period{
				Start: time.Date(ref.Year(), 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(ref.Year(), 6, 30, 0, 0, 0, 0, time.UTC),
			}
		}
		return Period{
			Start: time.Date(ref.Year(), 7, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(ref.Year(), 12, 31, 0, 0, 0, 0, time.UTC),
		}
	case LevelMonth:
		start := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		return Period{Start: start, End: end}
	case LevelWeek:
		// ISO week: Monday → Sunday
		offset := int(ref.Weekday())
		if offset == 0 {
			offset = 7
		}
		start := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, time.UTC).
			AddDate(0, 0, -(offset - 1))
		end := start.AddDate(0, 0, 6)
		return Period{Start: start, End: end}
	}
	return Period{}
}

// CurrentWeek returns the Mon-Sun period containing today (UTC).
func (c Cadence) CurrentWeek(now time.Time) Period { return c.Compute(LevelWeek, now) }

// CurrentMonth returns the calendar-month period containing today.
func (c Cadence) CurrentMonth(now time.Time) Period { return c.Compute(LevelMonth, now) }
