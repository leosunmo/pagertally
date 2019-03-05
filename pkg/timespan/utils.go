package timespan

import (
	"sort"
	"time"
)

// FlattenDate returns the timestamp without hours, mins or seconds.
// This is because time.Time.Truncate() doesn't work with non-UTC time
func FlattenDate(t time.Time) time.Time {
	y, m, d := t.Date()
	loc := t.Location()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// FlattenTime returns the timestamp without minutes or seconds.
// This is because time.Time.Truncate() doesn't work with non-UTC time
func FlattenTime(t time.Time) time.Time {
	y, m, d := t.Date()
	h, _, _ := t.Clock()
	return time.Date(y, m, d, h, 0, 0, 0, t.Location())
}

// AdjustForTimezone takes a timestamp and adds the offset of the
// provided timezone location and then returns the timestamp
// with the offset added/removed presented in the correct timezone
func AdjustForTimezone(t time.Time, loc *time.Location) time.Time {
	flatTime := FlattenTime(t)
	_, tzOffsetSeconds := flatTime.In(loc).Zone()
	return flatTime.Add(time.Second * time.Duration(-tzOffsetSeconds)).In(loc)
}

// Deduplicate returns a de-duplicated slice of span
func Deduplicate(spans []Span) []Span {
	result := []Span{}
	sort.Sort(Spans(spans))
	if len(spans) != 0 {
		j := 0
		for i := 1; i < len(spans); i++ {
			if spans[j].Equal(spans[i]) {
				continue
			}
			j++
			// preserve the original data
			// in[i], in[j] = in[j], in[i]
			// only set what is required
			spans[j] = spans[i]
		}
		result = spans[:j+1]
	}
	return result
}

// EndOfDay returns the end of the day (23:59) for the provided time
func EndOfDay(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, day.Location())
}

// StartOfDay returns the start of the day (00:00) for the provided time
func StartOfDay(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
}

// IsEndOfDay returns true if the timestamp is at 23:59:59
func IsEndOfDay(day time.Time) bool {
	return day.Hour() == 23 && day.Minute() == 59 && day.Second() == 59
}

// IsStartOfDay returns true if the timestamp is at 0:0:0
func IsStartOfDay(day time.Time) bool {
	return day.Hour() == 0 && day.Minute() == 0 && day.Second() == 0
}

// MergeSpans tries to merge as many bordering spans as possible
func MergeSpans(spans Spans) []Span {
	mergedSpans := []Span{}
	sort.Sort(spans)
	latestEnd := time.Time{}
	for i, span := range spans {
		if len(spans) == i+1 {
			if span.end.Before(latestEnd) || span.end.Equal(latestEnd) {
				break
			} else {
				latestEnd = span.end
			}
		} else {
			if span.end.Before(latestEnd) || span.end.Equal(latestEnd) {
				continue
			}
			latestEnd = FindLastEnd(spans[i:])
		}
		mergedSpan := New(span.start, latestEnd)
		mergedSpans = append(mergedSpans, mergedSpan)
	}
	return mergedSpans
}

// FindLastEnd starts from index 0 and tries to find the last end time of all consecutively bordering spans
func FindLastEnd(spans []Span) time.Time {
	latestEnd := spans[0].end
	for i := 0; len(spans) != i+1; i++ {
		latestEnd = spans[i].end
		if spans[i].Borders(spans[i+1]) {
			latestEnd = spans[i+1].end
		} else {
			return latestEnd
		}
	}
	return latestEnd
}
