// Modfied version of SaidinWoT's timespan library, https://github.com/SaidinWoT/timespan

package timespan

import (
	"time"

	timerange "github.com/leosunmo/timerange-go"
)

const spanDateFormat = "20060102"

//Span represents an inclusive range between two time instants.
//
//The zero value of type span has both start and end times set to the zero value
//of type Time. The zero value is returned by the Intersection and Gap methods
//when there is no span fitting their purposes.
type Span struct {
	start, end time.Time
}
type AttributedSpan struct {
	Span
	SpanType OnCallAttribute
}
type AttributedSpans []AttributedSpan

type Spans []Span

// ScheduleName is the name of a Pagerduty Schedule
type ScheduleName string

// ScheduleUserShifts is a map of UserShifts by the schedule name
type ScheduleUserShifts map[ScheduleName]UserShifts

// User is a Pagerduty User
type User struct {
	Name     string
	Location *time.Location
}

type UserShiftResults struct {
	User      User
	Schedule  ScheduleName
	Shifts    []Span
	Breakdown AttributedSpans
}

// UserShifts is a map of users to slice of their shifts
type UserShifts map[User][]Span

// OnCallAttribute determines what type of span it is,
// bussiness hours, out of office, stat holiday etc.
type OnCallAttribute int

const (
	Unknown OnCallAttribute = iota
	// Business is a span within business hours, e.g. 9:00 to 18:00
	Business // 1
	// AfterHours is a span after normal business hours, e.g. 18:00 to 9:00
	AfterHours // 2
	// Weekend is a span within a weekend, e.g. 18:00 Friday until 9:00 Monday
	Weekend // 3
	// StatHoliday is a span within statutory holidays, 00:00 to 00:00 e.g. Christmas Eve
	StatHoliday // 4
	// CompanyDay is a span within a company mandated day-off, 00:00 to 00:00 e.g. Christmas Party
	CompanyDay // 5
)

// TotalShifts returns the total number of shifts
func (u UserShiftResults) TotalShifts() int {
	return len(u.Shifts)
}

// BusinessHoursDur returns duration of business hours on call
func (spans AttributedSpans) BusinessHoursDur() time.Duration {
	var durs time.Duration
	for _, span := range spans {
		if span.SpanType == Business {
			durs += span.Duration()
		}
	}
	return durs
}

// AfterHoursDur returns duration of afterhours on call
func (spans AttributedSpans) AfterHoursDur() time.Duration {
	var durs time.Duration
	for _, span := range spans {
		if span.SpanType == AfterHours {
			durs += span.Duration()
		}
	}
	return durs
}

// WeekendDur returns duration of weekend hours on call
func (spans AttributedSpans) WeekendDur() time.Duration {
	var durs time.Duration
	for _, span := range spans {
		if span.SpanType == Weekend {
			durs += span.Duration()
		}
	}
	return durs
}

// StatDur returns duration of statday holiday hours on call
func (spans AttributedSpans) StatDur() time.Duration {
	var durs time.Duration
	for _, span := range spans {
		if span.SpanType == StatHoliday {
			durs += span.Duration()
		}
	}
	return durs
}

// CompanyDayDur returns duration of company days holiday hours on call
func (spans AttributedSpans) CompanyDayDur() time.Duration {
	var durs time.Duration
	for _, span := range spans {
		if span.SpanType == CompanyDay {
			durs += span.Duration()
		}
	}
	return durs
}

// CompanyDayCount returns duration of company days holiday hours on call
func (spans AttributedSpans) CompanyDayCount() int {
	var count int
	for _, span := range spans {
		if span.SpanType == CompanyDay {
			count++
		}
	}
	return count
}

// TotalDur returns the total duration on call
func (spans AttributedSpans) TotalDur() time.Duration {
	var totalDur time.Duration
	for _, span := range spans {
		dur := span.Duration()
		totalDur += dur
	}
	return totalDur
}

func (spans AttributedSpans) Less(a, b int) bool {
	return spans[a].start.Before(spans[b].start)
}

func (spans AttributedSpans) Swap(a, b int) {
	spans[a], spans[b] = spans[b], spans[a]
}

func (spans AttributedSpans) Len() int {
	return len(spans)
}

//Start returns the time instant at the start of s.
func (as AttributedSpan) Start() time.Time {
	return as.Span.Start()
}

//End returns the time instant at the end of s.
func (as AttributedSpan) End() time.Time {
	return as.Span.End()
}

// New creates a new span from start time to end time.
// If end time is before start, they will be flipped around.
func New(s time.Time, e time.Time) Span {
	start := s
	end := e
	if end.Before(s) {
		start, end = end, start
	}

	return Span{
		start: start,
		end:   end,
	}
}

func (spans Spans) Less(a, b int) bool {
	return spans[a].start.Before(spans[b].start)
}

func (spans Spans) Swap(a, b int) {
	spans[a], spans[b] = spans[b], spans[a]
}

func (spans Spans) Len() int {
	return len(spans)
}

//Start returns the time instant at the start of s.
func (s Span) Start() time.Time {
	return s.start
}

//End returns the time instant at the end of s.
func (s Span) End() time.Time {
	return s.end
}

//Duration returns the length of time represented by s.
func (s Span) Duration() time.Duration {
	return s.end.Sub(s.start)
}

//After reports whether s begins after t.
func (s Span) After(t time.Time) bool {
	return s.start.After(t)
}

//Before reports whether s ends before t.
func (s Span) Before(t time.Time) bool {
	return s.end.Before(t)
}

//Borders reports whether s and r are contiguous time intervals.
func (s Span) Borders(r Span) bool {
	return s.start.Equal(r.end) || s.end.Equal(r.start)
}

//ContainsTime reports whether t is within s.
func (s Span) ContainsTime(t time.Time) bool {
	return !(t.Before(s.start) || t.After(s.end))
}

//Contains reports whether r is entirely within s.
func (s Span) Contains(r Span) bool {
	return (s.ContainsTime(r.start) || s.start == r.start) && (s.ContainsTime(r.end) || s.end == r.end)
}

//Encompass returns the minimum span that fully contains both r and s.
func (s Span) Encompass(r Span) Span {
	return Span{
		start: tmin(s.start, r.start),
		end:   tmax(s.end, r.end),
	}
}

//Equal reports whether s and r represent the same time intervals, ignoring
//the locations of the times.
func (s Span) Equal(r Span) bool {
	return s.start.Equal(r.start) && s.end.Equal(r.end)
}

//Follows reports whether s begins after or at the end of r.
func (s Span) Follows(r Span) bool {
	return !s.start.Before(r.end)
}

//Gap returns a span corresponding to the period between s and r.
//If s and r have a non-zero overlap, a zero span is returned.
func (s Span) Gap(r Span) Span {
	if s.Overlaps(r) {
		return Span{}
	}
	return Span{
		start: tmin(s.end, r.end),
		end:   tmax(s.start, r.start),
	}
}

//Intersection returns both a span corresponding to the non-zero overlap of
//s and r and a bool indicating whether such an overlap existed.
//If s and r do not overlap, a zero span is returned with false.
func (s Span) Intersection(r Span) (Span, bool) {
	if s.Equal(r) {
		return s, true
	}
	if !s.Overlaps(r) {
		return Span{}, false
	}
	return Span{
		start: tmax(s.start, r.start),
		end:   tmin(s.end, r.end),
	}, true
}

//IsZero reports whether s represents the zero-length span starting and ending
//on January 1, year 1, 00:00:00 UTC.
func (s Span) IsZero() bool {
	return s.start.IsZero() && s.end.IsZero()
}

//Offset returns s with its start time offset by d. It is equivalent to
//Newspan(s.Start().Add(d), s.Duration()).
func (s Span) Offset(d time.Duration) Span {
	return Span{
		start: s.start.Add(d),
		end:   s.end.Add(d),
	}
}

//OffsetDate returns s with its start time offset by the given years, months,
//and days. It is equivalent to
//Newspan(s.Start().AddDate(years, months, days), s.Duration()).
func (s Span) OffsetDate(years, months, days int) Span {
	d := s.Duration()
	t := s.start.AddDate(years, months, days)
	return Span{
		start: t,
		end:   t.Add(d),
	}
}

//Overlaps reports whether s and r intersect for a non-zero duration.
func (s Span) Overlaps(r Span) bool {
	return s.start.Before(r.end) && s.end.After(r.start)
}

// TrimIfOverlaps returns span s cut off to not overlap with r, from either direction.
// Returns s and false if no overlap.
// Returns a zero len span and true if s fits within r
func (s Span) TrimIfOverlaps(r Span) (Span, bool) {
	// Check if they are a perfect match
	if r.Equal(s) {
		return Span{}, true
	}
	// Only do anything if they actually overlap
	if s.Overlaps(r) {
		// r overlaps s from the end
		if s.start.Before(r.start) && !s.end.Equal(r.start) {
			n := New(s.start, r.start)
			return n, true
		}
		// r overlaps s from the beginning
		if s.start.After(r.start) && !s.start.Equal(r.end) {
			n := New(r.end, s.end)
			return n, true
		}
		// s fits inside r
		if s.Contains(r) {
			return Span{}, true
		}
		if s.start.Equal(r.start) {
			n := New(r.end, s.end)
			return n, true
		}

	}
	return s, false
}

//Precedes reports whether s ends before or at the start of r.
func (s Span) Precedes(r Span) bool {
	return !s.end.After(r.start)
}

// Dates returns all dates (flattened time, only dates) in the span as a slice of time.Time
func (s Span) Dates() []time.Time {
	var dates []time.Time
	dedupeDates := make(map[string]struct{})
	dayIter := timerange.New(s.start, s.end, 23*time.Hour, true)
	dedupeDates[dayIter.Current().Format(spanDateFormat)] = struct{}{}
	for dayIter.Next() {
		dedupeDates[dayIter.Current().Format(spanDateFormat)] = struct{}{}
	}
	dedupeDates[s.end.Format(spanDateFormat)] = struct{}{}
	for stringDate := range dedupeDates {
		date, err := time.Parse(spanDateFormat, stringDate)
		if err != nil {
			// TODO: Problably need real error management here
			panic(err)
		}
		dates = append(dates, date)
	}
	return dates
}

// SplitByDay returns a slice of span split by day from the provided span
// If the end of the span is exactly midnight the next day it will be truncated
func (s Span) SplitByDay() []Span {
	splitSpans := []Span{}

	// Strip time from start and end dates for comparisons
	strippedStart := time.Date(
		s.start.Year(),
		s.start.Month(),
		s.start.Day(),
		0, 0, 0, 0,
		s.start.Location())
	strippedEnd := time.Date(
		s.end.Year(),
		s.end.Month(),
		s.end.Day(),
		0, 0, 0, 0,
		s.end.Location())

	// If the year,month or day (and location?) are not the same, split it.
	if !strippedStart.Equal(strippedEnd) {
		// Special case if the span ends at midnight the next day exactly
		if s.end.Equal(strippedStart.AddDate(0, 0, 1)) {
			// truncate so that the span ends 23:59 the same day as start.
			newSpan := New(s.start, EndOfDay(s.start))
			splitSpans = append(splitSpans, newSpan)
			return splitSpans
		}
		timeDates := s.Dates()
		for i, date := range timeDates {
			if i < 1 {
				// If it's the first iteration, create a span from the
				// start and EndOfDay of the original span
				newSpan := New(s.start, EndOfDay(s.start))
				splitSpans = append(splitSpans, newSpan)
				continue
			} else if i == (len(timeDates) - 1) {
				// If it's the last iteration, create a span from the
				// StartOfDay and end of the original span
				newSpan := New(StartOfDay(s.end), s.end)
				splitSpans = append(splitSpans, newSpan)
				continue
			}
			// We're not at the beginning or end of the loop
			newSpan := New(StartOfDay(date), EndOfDay(date))
			splitSpans = append(splitSpans, newSpan)
		}
	}
	return splitSpans
}

// SetAttribute adds an on-call attribute to the span
func (as AttributedSpan) SetAttribute(attrib OnCallAttribute) {
	as.SpanType = attrib
}

// Attribute returns the OnCallAttribute of the span
func (as AttributedSpan) Attribute() OnCallAttribute {
	return as.SpanType
}

//tmax returns the later of two time instants.
func tmax(t, u time.Time) time.Time {
	if t.After(u) {
		return t
	}
	return u
}

//tmin returns the earlier of two time instants.
func tmin(t, u time.Time) time.Time {
	if t.Before(u) {
		return t
	}
	return u
}
