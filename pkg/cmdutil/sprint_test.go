package cmdutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/raksul/go-clickup/clickup"
	"github.com/stretchr/testify/assert"
)

func TestParseMSTimestamp(t *testing.T) {
	t.Run("valid timestamp", func(t *testing.T) {
		result := ParseMSTimestamp("1736899200000")
		assert.False(t, result.IsZero())
		assert.Equal(t, time.UnixMilli(1736899200000), result)
	})

	t.Run("empty string", func(t *testing.T) {
		result := ParseMSTimestamp("")
		assert.True(t, result.IsZero())
	})

	t.Run("zero string", func(t *testing.T) {
		result := ParseMSTimestamp("0")
		assert.True(t, result.IsZero())
	})

	t.Run("invalid string", func(t *testing.T) {
		result := ParseMSTimestamp("not-a-number")
		assert.True(t, result.IsZero())
	})
}

// Real sprint timestamps from the workspace (AEST/UTC+10 midnight boundaries):
//
//	Sprint 85: start=1771250400000 (Tue 17 Feb 01:00 AEDT) due=1772459999999 (Tue 3 Mar 00:59:59.999 AEDT)
//	Sprint 86: start=1772460000000 (Tue 3 Mar 01:00 AEDT)  due=1773669599999 (Tue 17 Mar 00:59:59.999 AEDT)
//
// ClickUp stores dates as midnight in AEST (UTC+10), so:
//   - start = Mon 14:00:00.000 UTC (= Tue 00:00 AEST)
//   - due   = Mon 13:59:59.999 UTC (= Mon 23:59:59.999 AEST, i.e. end of the last day)
//
// Sprints are contiguous with a 1ms gap between due and next start.

func sprintLists() []clickup.List {
	return []clickup.List{
		{ID: "sprint-85", Name: "Sprint 85", StartDate: "1771250400000", DueDate: "1772459999999"},
		{ID: "sprint-86", Name: "Sprint 86", StartDate: "1772460000000", DueDate: "1773669599999"},
	}
}

// utcTime is a helper to construct a time in UTC.
func utcTime(year, month, day, hour, min, sec int) time.Time {
	return time.Date(year, time.Month(month), day, hour, min, sec, 0, time.UTC)
}

// aestTime constructs a time in AEST (UTC+10), the timezone ClickUp uses for boundaries.
func aestTime(year, month, day, hour, min, sec int) time.Time {
	aest := time.FixedZone("AEST", 10*60*60)
	return time.Date(year, time.Month(month), day, hour, min, sec, 0, aest)
}

// aedtTime constructs a time in AEDT (UTC+11), the actual local timezone during these sprints.
func aedtTime(year, month, day, hour, min, sec int) time.Time {
	aedt := time.FixedZone("AEDT", 11*60*60)
	return time.Date(year, time.Month(month), day, hour, min, sec, 0, aedt)
}

// nzdtTime constructs a time in NZDT (UTC+13).
func nzdtTime(year, month, day, hour, min, sec int) time.Time {
	nzdt := time.FixedZone("NZDT", 13*60*60)
	return time.Date(year, time.Month(month), day, hour, min, sec, 0, nzdt)
}

func TestMatchSprintListID(t *testing.T) {
	lists := sprintLists()

	t.Run("day before sprint 85 starts", func(t *testing.T) {
		// Mon 16 Feb 2026 12:00 UTC — before Sprint 85 start (Mon 16 Feb 14:00 UTC)
		now := utcTime(2026, 2, 16, 12, 0, 0)
		assert.Equal(t, "", MatchSprintListID(lists, now))
	})

	t.Run("exact start of sprint 85", func(t *testing.T) {
		// Mon 16 Feb 2026 14:00:00.000 UTC = Sprint 85 start
		now := time.UnixMilli(1771250400000)
		assert.Equal(t, "sprint-85", MatchSprintListID(lists, now))
	})

	t.Run("1 day into sprint 85", func(t *testing.T) {
		now := utcTime(2026, 2, 17, 14, 0, 0)
		assert.Equal(t, "sprint-85", MatchSprintListID(lists, now))
	})

	t.Run("2 days into sprint 85", func(t *testing.T) {
		now := utcTime(2026, 2, 18, 14, 0, 0)
		assert.Equal(t, "sprint-85", MatchSprintListID(lists, now))
	})

	t.Run("midway through sprint 85", func(t *testing.T) {
		now := utcTime(2026, 2, 24, 10, 0, 0)
		assert.Equal(t, "sprint-85", MatchSprintListID(lists, now))
	})

	t.Run("last day of sprint 85 morning UTC", func(t *testing.T) {
		// Mon 2 Mar 2026 10:00 UTC — still within Sprint 85 (due is 13:59:59.999 UTC)
		now := utcTime(2026, 3, 2, 10, 0, 0)
		assert.Equal(t, "sprint-85", MatchSprintListID(lists, now))
	})

	t.Run("exact end of sprint 85", func(t *testing.T) {
		// Mon 2 Mar 2026 13:59:59.999 UTC = Sprint 85 due
		now := time.UnixMilli(1772459999999)
		assert.Equal(t, "sprint-85", MatchSprintListID(lists, now))
	})

	t.Run("1ms after sprint 85 ends is sprint 86 start", func(t *testing.T) {
		// Mon 2 Mar 2026 14:00:00.000 UTC = Sprint 86 start (1ms after Sprint 85 due)
		now := time.UnixMilli(1772460000000)
		assert.Equal(t, "sprint-86", MatchSprintListID(lists, now))
	})

	t.Run("1 day into sprint 86", func(t *testing.T) {
		now := utcTime(2026, 3, 3, 14, 0, 0)
		assert.Equal(t, "sprint-86", MatchSprintListID(lists, now))
	})

	t.Run("2 days into sprint 86", func(t *testing.T) {
		now := utcTime(2026, 3, 4, 14, 0, 0)
		assert.Equal(t, "sprint-86", MatchSprintListID(lists, now))
	})

	t.Run("exact end of sprint 86", func(t *testing.T) {
		now := time.UnixMilli(1773669599999)
		assert.Equal(t, "sprint-86", MatchSprintListID(lists, now))
	})

	t.Run("after sprint 86 ends", func(t *testing.T) {
		now := time.UnixMilli(1773669600000)
		assert.Equal(t, "", MatchSprintListID(lists, now))
	})
}

func TestMatchSprintListID_MidnightBoundaries(t *testing.T) {
	lists := sprintLists()

	// Sprint boundaries are at AEST midnight (UTC+10).
	// In AEDT (UTC+11, actual local time), that's 01:00.
	// These tests verify correct matching at midnight in various timezones.

	t.Run("AEST midnight start of sprint 86", func(t *testing.T) {
		// Tue 3 Mar 2026 00:00:00 AEST = Mon 2 Mar 14:00:00 UTC = Sprint 86 start
		now := aestTime(2026, 3, 3, 0, 0, 0)
		assert.Equal(t, "sprint-86", MatchSprintListID(lists, now))
	})

	t.Run("AEST 23:59:59 last day of sprint 85", func(t *testing.T) {
		// Mon 2 Mar 2026 23:59:59 AEST = Mon 2 Mar 13:59:59 UTC — within Sprint 85
		now := aestTime(2026, 3, 2, 23, 59, 59)
		assert.Equal(t, "sprint-85", MatchSprintListID(lists, now))
	})

	t.Run("AEDT midnight on sprint 86 start date", func(t *testing.T) {
		// Tue 3 Mar 2026 00:00:00 AEDT = Mon 2 Mar 13:00:00 UTC
		// This is BEFORE Sprint 86 start (14:00 UTC) and WITHIN Sprint 85 (due 13:59:59.999 UTC)
		now := aedtTime(2026, 3, 3, 0, 0, 0)
		assert.Equal(t, "sprint-85", MatchSprintListID(lists, now),
			"AEDT midnight on the sprint start date is still in the previous sprint — "+
				"ClickUp uses AEST boundaries, so AEDT is 1 hour early")
	})

	t.Run("AEDT 01:00 on sprint 86 start date", func(t *testing.T) {
		// Tue 3 Mar 2026 01:00:00 AEDT = Mon 2 Mar 14:00:00 UTC = Sprint 86 start
		now := aedtTime(2026, 3, 3, 1, 0, 0)
		assert.Equal(t, "sprint-86", MatchSprintListID(lists, now))
	})

	t.Run("NZDT midnight on sprint 86 start date", func(t *testing.T) {
		// Tue 3 Mar 2026 00:00:00 NZDT = Mon 2 Mar 11:00:00 UTC — within Sprint 85
		now := nzdtTime(2026, 3, 3, 0, 0, 0)
		assert.Equal(t, "sprint-85", MatchSprintListID(lists, now),
			"NZDT midnight is 3 hours before the AEST boundary")
	})

	t.Run("UTC midnight between sprints", func(t *testing.T) {
		// Tue 3 Mar 2026 00:00:00 UTC = still within Sprint 85 (due Mon 2 Mar 13:59:59.999 UTC)
		// Wait — Mon 2 Mar was the last day. Tue 3 Mar 00:00 UTC is 10:00 AEST Tue = in Sprint 86
		// Let's verify: 3 Mar 00:00 UTC = 1772611200000 ms
		now := utcTime(2026, 3, 3, 0, 0, 0)
		assert.Equal(t, "sprint-86", MatchSprintListID(lists, now),
			"UTC midnight on Mar 3 is well into Sprint 86 (which started Mon 2 Mar 14:00 UTC)")
	})
}

func TestMatchSprintListID_TimezoneConsistency(t *testing.T) {
	lists := sprintLists()

	// The same instant in time should resolve to the same sprint regardless of
	// which timezone the time.Time is constructed in.
	t.Run("same instant different timezones all match same sprint", func(t *testing.T) {
		// Pick an instant solidly within Sprint 86: Wed 4 Mar 2026 00:00 UTC
		instants := []time.Time{
			utcTime(2026, 3, 4, 0, 0, 0),
			aestTime(2026, 3, 4, 10, 0, 0),  // same instant
			aedtTime(2026, 3, 4, 11, 0, 0),  // same instant
			nzdtTime(2026, 3, 4, 13, 0, 0),  // same instant
		}

		for _, now := range instants {
			assert.Equal(t, "sprint-86", MatchSprintListID(lists, now),
				"timezone: %s", now.Location().String())
		}
	})

	t.Run("boundary instant same in all timezones", func(t *testing.T) {
		// The exact Sprint 86 start instant expressed in different zones
		startMS := int64(1772460000000)
		instants := []time.Time{
			time.UnixMilli(startMS).UTC(),
			time.UnixMilli(startMS).In(time.FixedZone("AEST", 10*60*60)),
			time.UnixMilli(startMS).In(time.FixedZone("AEDT", 11*60*60)),
			time.UnixMilli(startMS).In(time.FixedZone("NZDT", 13*60*60)),
		}

		for _, now := range instants {
			assert.Equal(t, "sprint-86", MatchSprintListID(lists, now),
				"timezone: %s", now.Location().String())
		}
	})
}

func TestMatchSprintListID_NextSprintExistsBeforeCurrentEnds(t *testing.T) {
	// Scenario: Sprint 86 is still active, but Sprint 87 has already been created
	// with future dates. MatchSprintListID should return the current sprint.
	lists := []clickup.List{
		{ID: "sprint-85", Name: "Sprint 85", StartDate: "1771250400000", DueDate: "1772459999999"},
		{ID: "sprint-86", Name: "Sprint 86", StartDate: "1772460000000", DueDate: "1773669599999"},
		{ID: "sprint-87", Name: "Sprint 87", StartDate: "1773669600000", DueDate: "1774879199999"},
	}

	t.Run("mid sprint 86 with future sprint 87 existing", func(t *testing.T) {
		now := utcTime(2026, 3, 10, 12, 0, 0) // midway through Sprint 86
		assert.Equal(t, "sprint-86", MatchSprintListID(lists, now))
	})

	t.Run("last moment of sprint 86 with sprint 87 existing", func(t *testing.T) {
		now := time.UnixMilli(1773669599999)
		assert.Equal(t, "sprint-86", MatchSprintListID(lists, now))
	})

	t.Run("first moment of sprint 87", func(t *testing.T) {
		now := time.UnixMilli(1773669600000)
		assert.Equal(t, "sprint-87", MatchSprintListID(lists, now))
	})
}

func TestMatchSprintListID_OverlappingSprints(t *testing.T) {
	// Edge case: two sprints with overlapping date ranges.
	// The function returns the first match in list order.
	lists := []clickup.List{
		{ID: "sprint-86", Name: "Sprint 86", StartDate: "1772460000000", DueDate: "1773669599999"},
		{ID: "sprint-86b", Name: "Sprint 86b", StartDate: "1773060000000", DueDate: "1774269599999"},
	}

	t.Run("in overlap period returns first match", func(t *testing.T) {
		// Pick a time in the overlap zone
		now := time.UnixMilli(1773200000000)
		result := MatchSprintListID(lists, now)
		assert.Equal(t, "sprint-86", result, "returns the first matching list in order")
	})
}

func TestMatchSprintListID_EdgeCases(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		assert.Equal(t, "", MatchSprintListID(nil, time.Now()))
	})

	t.Run("lists without dates", func(t *testing.T) {
		lists := []clickup.List{
			{ID: "backlog", Name: "Backlog", StartDate: "", DueDate: ""},
			{ID: "no-start", Name: "No Start", StartDate: "", DueDate: "1773669599999"},
			{ID: "no-due", Name: "No Due", StartDate: "1772460000000", DueDate: ""},
		}
		assert.Equal(t, "", MatchSprintListID(lists, time.Now()))
	})

	t.Run("mixed lists with and without dates", func(t *testing.T) {
		lists := []clickup.List{
			{ID: "backlog", Name: "Backlog", StartDate: "", DueDate: ""},
			{ID: "sprint-86", Name: "Sprint 86", StartDate: "1772460000000", DueDate: "1773669599999"},
		}
		now := utcTime(2026, 3, 10, 12, 0, 0)
		assert.Equal(t, "sprint-86", MatchSprintListID(lists, now))
	})
}

func TestMatchSprintListID_WalkThroughSprintDays(t *testing.T) {
	lists := sprintLists()

	// Walk through each day of Sprint 86 (Mar 3-16 in AEST) at midday AEST
	// to verify consistent matching across the entire sprint duration.
	for day := 3; day <= 16; day++ {
		t.Run(fmt.Sprintf("Mar %d midday AEST", day), func(t *testing.T) {
			now := aestTime(2026, 3, day, 12, 0, 0)
			assert.Equal(t, "sprint-86", MatchSprintListID(lists, now),
				"should match Sprint 86 on Mar %d", day)
		})
	}

	// Day before (Mar 2) should be Sprint 85
	t.Run("Mar 2 midday AEST is sprint 85", func(t *testing.T) {
		now := aestTime(2026, 3, 2, 12, 0, 0)
		assert.Equal(t, "sprint-85", MatchSprintListID(lists, now))
	})

	// Day after (Mar 17) should be no match
	t.Run("Mar 17 midday AEST is no match", func(t *testing.T) {
		now := aestTime(2026, 3, 17, 12, 0, 0)
		assert.Equal(t, "", MatchSprintListID(lists, now))
	})
}
