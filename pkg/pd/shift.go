package pd

import (
	"time"

	"github.com/leosunmo/pagerduty-schedule/pkg/calendar"

	timerange "github.com/leosunmo/timerange-go"
)

// ScheduleUserShifts is a map of UserShifts by the schedule name
type ScheduleUserShifts map[string]UserShifts

// UserShifts is a map of users to slice of their shifts
type UserShifts map[string][]Shift

// Shift is a single shift by one user and the number of afterhours or stat time spent in hours.
type Shift struct {
	StartDate    time.Time
	EndDate      time.Time
	Duration     time.Duration
	ScheduleName string
	// ShiftHours is the individual hours within a shift
	// The map Key is a timestamp of the beginning of the hour
	// 12:00:00 is the hour between 12:00:00 and 13:00:00

	// The map Value is an int that represents the classification of
	// the hour, such as business time, afterhour, stat day etc.
	ShiftHours map[time.Time]int

	Calendar *calendar.Calendar
}

// ProcessHours iterates through every hour of the shift and matches it
// against stat holidays, weekends etc.
// Flattens the start time the beginning of the hour,
// rounds the end date to the nearest hour
func (s *Shift) ProcessHours() {
	if s.Duration < time.Minute*30 {
		return
	}
	if s.Duration < time.Hour {
		s.ShiftHours[s.StartDate] = s.Calendar.GetHourTag(s.StartDate)
		return
	}
	start := s.StartDate.Round(time.Hour)
	end := s.EndDate.Add(time.Duration(-31) * time.Minute).Round(time.Hour)
	tr := timerange.New(start, end, time.Hour)
	for tr.Next() {
		s.ShiftHours[tr.Current()] = s.Calendar.GetHourTag(tr.Current())
	}
}
