package datasources

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/leosunmo/pagertally/pkg/timespan"

	"github.com/leosunmo/pagertally/pkg/config"
)

const (
	timeFormat = "2006-01-02T15:04:05"
	firstJan   = "2019-01-01T00:00:00"
	lastJan    = "2019-01-31T00:00:00"
)

var aklTz, _ = time.LoadLocation("Pacific/Auckland")
var nyTz, _ = time.LoadLocation("America/New_York")

func TestCompanyDaySpans(t *testing.T) {
	var err error
	start, err := time.ParseInLocation(timeFormat, firstJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}
	end, err := time.ParseInLocation(timeFormat, lastJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	config.GlobalConfig = config.Config{
		ScheduleSpan: timespan.New(start, end), // Company days currently ignores any time limits
		Timezone:     "Pacific/Auckland",
		CompanyDays:  []string{"15/01/2019", "23/01/2019", "31/01/2019", "01/04/2019"},
	}

	vds := NewCompanyDayDataSource()

	if len(vds.Spans()) != 4 {
		t.Errorf("Should have 4 company day spans, got %d", len(vds.Spans()))
	}
}
func TestCalendarSpans(t *testing.T) {
	var err error
	start, err := time.ParseInLocation(timeFormat, firstJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}
	end, err := time.ParseInLocation(timeFormat, lastJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	config.GlobalConfig = config.Config{
		Holidays:     []string{"Auckland Anniversary", "Wellington Anniversary"},
		CalendarURL:  "http://apps.employment.govt.nz/ical/public-holidays-all.ics",
		Timezone:     "Pacific/Auckland",
		ScheduleSpan: timespan.New(start, end),
	}

	cal := NewCalendarDataSource()

	if len(cal.Spans()) != 2 {
		t.Errorf("Calendar should contain 2 spans, got %d", len(cal.Spans()))
	}
}

func TestCommonWeekendsSpans(t *testing.T) {
	var err error
	start, err := time.ParseInLocation(timeFormat, firstJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}
	end, err := time.ParseInLocation(timeFormat, lastJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	config.GlobalConfig = config.Config{
		ScheduleSpan: timespan.New(start, end),
		BusinessHours: config.BusinessHoursStruct{
			Start: "08:00",
			End:   "17:30",
		},
		Timezone: "Pacific/Auckland",
	}

	wkds := NewWeekendDataSource()
	if len(wkds.Spans()) != 4 {
		t.Errorf("Weekend datasource should contain 4 spans, got %d", len(wkds.Spans()))
	}
}

func TestCommonAfterhoursSpans(t *testing.T) {
	var err error
	start, err := time.ParseInLocation(timeFormat, firstJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}
	end, err := time.ParseInLocation(timeFormat, lastJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	config.GlobalConfig = config.Config{
		ScheduleSpan: timespan.New(start, end),
		BusinessHours: config.BusinessHoursStruct{
			Start: "08:00",
			End:   "17:30",
		},
	}

	oobds := NewAfterHoursDataSource()
	spans := oobds.Spans()
	if len(spans) != 20 {
		t.Errorf("Expected 20 spans, got %d\nFirst span: %s to %s\nLast span: %s to %s\n", len(spans), spans[0].Start(), spans[0].End(), spans[len(spans)-1].Start(), spans[len(spans)-1].End())
	}
}

func TestWeekendSpanTimestamps(t *testing.T) {
	var err error

	var businessStartHour = 8
	var businessStartMinute = 0
	var businessEndHour = 17
	var businessEndMinute = 30

	start, err := time.ParseInLocation(timeFormat, firstJan, aklTz)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}
	end, err := time.ParseInLocation(timeFormat, lastJan, aklTz)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	config.GlobalConfig = config.Config{
		ScheduleSpan: timespan.New(start, end),
		BusinessHours: config.BusinessHoursStruct{
			Start: "08:00",
			End:   "17:30",
		},
		Timezone: "Pacific/Auckland",
	}

	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	wkds := NewWeekendDataSource()
	spans := wkds.Spans()
	for _, span := range spans {
		switch span.Start().Weekday() {
		case time.Friday:
			if span.Start().Hour() != businessEndHour || span.Start().Minute() != businessEndMinute {
				t.Errorf("Expected Friday start hour to be %d and minute to be %d (business day end)\nGot %s to %s", businessEndHour, businessEndMinute, span.Start(), span.End())
			}
			if span.End().Weekday() != time.Monday || span.End().Hour() != businessStartHour || span.End().Minute() != businessStartMinute {
				t.Errorf("Expected Friday end hour to be %d and minute to be %d (Monday, start of business)\nGot %s to %s", businessStartHour, businessStartMinute, span.Start(), span.End())
			}
		default:
			t.Errorf("Weekend span starts on %s but should have started on Friday.", span.Start().Weekday())
		}
	}
}

func TestAfterHoursSpanTimestamps(t *testing.T) {
	var err error

	var businessStartHour = 8
	var businessStartMinute = 0
	var businessEndHour = 17
	var businessEndMinute = 30

	start, err := time.ParseInLocation(timeFormat, firstJan, aklTz)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}
	end, err := time.ParseInLocation(timeFormat, lastJan, aklTz)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	config.GlobalConfig = config.Config{
		ScheduleSpan: timespan.New(start, end),
		BusinessHours: config.BusinessHoursStruct{
			Start: "08:00",
			End:   "17:30",
		},
		Timezone: "Pacific/Auckland",
	}

	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	ahds := NewAfterHoursDataSource()
	spans := ahds.Spans()
	firstSpanDone := false
	if len(spans) != 19 {
		t.Errorf("Expected 19 spans, got %d\nFirst span: %s to %s\nLast span: %s to %s\n", len(spans), spans[0].Start(), spans[0].End(), spans[len(spans)-1].Start(), spans[len(spans)-1].End())
	}
	for _, span := range spans {
		switch span.Start().Weekday() {
		case time.Monday, time.Tuesday, time.Wednesday, time.Thursday:
			if span.Start().YearDay() == 1 || span.Start().YearDay() == 30 { // First of Jan is a Tueday and it's afterhours starts midnight since it's a new year
				if span.Start().Weekday() == time.Tuesday {
					if !firstSpanDone { // Only traverse this once since there's two spans starting on 1st Jan, only the first is special
						if span.Start().Weekday() != time.Tuesday || span.Start().Hour() != 0 || span.Start().Minute() != 0 {
							t.Errorf("Expected afterhours to start at midnight on %s, 1st Jan, %02d:%02d\nGot %s to %s", span.Start().Weekday(), 0, 0, span.Start(), span.End())
						}
						if span.End().Weekday() != time.Tuesday || span.End().Hour() != businessStartHour || span.End().Minute() != businessStartMinute {
							t.Errorf("Expected afterhours to end on %s at business start, %02d:%02d\nGot %s to %s", span.End().Weekday(), businessStartHour, businessStartMinute, span.Start(), span.End())
						}
						firstSpanDone = true
					}
				}
				if span.End().Weekday() == time.Thursday {
					if span.Start().Weekday() != time.Wednesday || span.Start().Hour() != businessEndHour || span.Start().Minute() != businessEndMinute {
						t.Errorf("Expected afterhours to start business dat end on %s, 30th of Jan, %02d:%02d\nGot %s to %s", span.Start().Weekday(), 0, 0, span.Start(), span.End())
					}
					if span.End().Weekday() != time.Thursday || span.End().Hour() != 0 || span.End().Minute() != 0 {
						t.Errorf("Expected afterhours to end on %s at midnight, %02d:%02d\nGot %s to %s", span.End().Weekday(), 0, 0, span.Start(), span.End())
					}
				}
			} else {
				if span.Start().Hour() != businessEndHour || span.Start().Minute() != businessEndMinute {
					t.Errorf("Expected afterhours to start at business day end on %s, %02d:%02d\nGot %s to %s", span.Start().Weekday(), businessEndHour, businessEndMinute, span.Start(), span.End())
				}
				if span.End().Weekday() != (span.Start().Weekday()+1) || span.End().Hour() != businessStartHour || span.End().Minute() != businessStartMinute {
					t.Errorf("Expected afterhours to end on %s at business start, %02d:%02d\nGot %s to %s", span.End().Weekday(), businessStartHour, businessStartMinute, span.Start(), span.End())
				}
			}
		default:
			t.Errorf("Didn't expect span start on weekday %s. Span: %s to %s", span.Start().Weekday(), span.Start(), span.End())
		}
	}
}

func TestCalendarSpanTimestamps(t *testing.T) {
	var err error
	start, err := time.ParseInLocation(timeFormat, firstJan, aklTz)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}
	end, err := time.ParseInLocation(timeFormat, lastJan, aklTz)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}
	holidays := []string{"Auckland Anniversary", "Wellington Anniversary", "Day after New Year's Day"}
	config.GlobalConfig = config.Config{
		ScheduleSpan: timespan.New(start, end),
		CalendarURL:  "http://apps.employment.govt.nz/ical/public-holidays-all.ics",
		Timezone:     "Pacific/Auckland",
		Holidays:     holidays,
	}

	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	cal := NewCalendarDataSource()
	spans := cal.Spans()
	testDate := time.Time{}
	for _, span := range spans {
		switch span.Start().Day() {
		case 2:
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-02T00:00:00", aklTz)
			if !span.Start().Equal(testDate) {
				t.Errorf("Expected Day after New Year's Day event to start %s\nGot %s to %s", testDate, span.Start(), span.End())
			}
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-03T00:00:00", aklTz)
			if !span.End().Equal(testDate) {
				t.Errorf("Expected Day after New Year's Day event to end %s\nGot %s to %s", testDate, span.Start(), span.End())
			}
		case 21:
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-21T00:00:00", aklTz)
			if !span.Start().Equal(testDate) {
				t.Errorf("Expected Wellington Anniversary event to start %s\nGot %s to %s", testDate, span.Start(), span.End())
			}
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-22T00:00:00", aklTz)
			if !span.End().Equal(testDate) {
				t.Errorf("Expected Wellington Anniversary event to end %s\nGot %s to %s", testDate, span.Start(), span.End())
			}
		case 28:
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-28T00:00:00", aklTz)
			if !span.Start().Equal(testDate) {
				t.Errorf("Expected Auckland Anniversary event to start %s\nGot %s to %s", testDate, span.Start(), span.End())
			}
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-29T00:00:00", aklTz)
			if !span.End().Equal(testDate) {
				t.Errorf("Expected Auckland Anniversary event to end %s\nGot %s to %s", testDate, span.Start(), span.End())
			}
		default:
			t.Errorf("Unknown event, date: %s\nExpected one of: %s ", span.Start(), strings.Join(holidays, ", "))
		}
	}
}

func TestCompanyDaySpansTimestamps(t *testing.T) {
	var err error
	start, err := time.ParseInLocation(timeFormat, firstJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}
	end, err := time.ParseInLocation(timeFormat, lastJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	config.GlobalConfig = config.Config{
		ScheduleSpan: timespan.New(start, end), // Company days currently ignores any time limits
		Timezone:     "Pacific/Auckland",
		CompanyDays:  []string{"15/01/2019", "23/01/2019", "31/01/2019", "01/04/2019"},
	}

	vds := NewCompanyDayDataSource()
	spans := vds.Spans()
	testDate := time.Time{}

	for _, span := range spans {
		switch span.Start().Weekday() {
		case time.Tuesday:
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-15T00:00:00", aklTz)
			if !span.Start().Equal(testDate) {
				t.Errorf("Expected the company day on 15/01/2019 to start midnight of the provided date\nGot %s to %s", span.Start(), span.End())
			}
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-16T00:00:00", aklTz)
			if !span.End().Equal(testDate) {
				t.Errorf("Expected the company day on 15/01/2019 to end midnight next day\nGot %s to %s", span.Start(), span.End())
			}
		case time.Wednesday:
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-23T00:00:00", aklTz)
			if !span.Start().Equal(testDate) {
				t.Errorf("Expected the company day on 23/01/2019 to start midnight of the provided date\nGot %s to %s", span.Start(), span.End())
			}
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-24T00:00:00", aklTz)
			if !span.End().Equal(testDate) {
				t.Errorf("Expected the company day on 23/01/2019 to end midnight next day\nGot %s to %s", span.Start(), span.End())
			}
		case time.Thursday:
			testDate, _ = time.ParseInLocation(timeFormat, "2019-01-31T00:00:00", aklTz)
			if !span.Start().Equal(testDate) {
				t.Errorf("Expected the company day on 31/01/2019 to start midnight of the provided date\nGot %s to %s", span.Start(), span.End())
			}
			testDate, _ = time.ParseInLocation(timeFormat, "2019-02-01T00:00:00", aklTz)
			if !span.End().Equal(testDate) {
				t.Errorf("Expected the company day on 31/01/2019 to end midnight next day\nGot %s to %s", span.Start(), span.End())
			}
		case time.Monday:
			testDate, _ = time.ParseInLocation(timeFormat, "2019-04-01T00:00:00", aklTz)
			if !span.Start().Equal(testDate) {
				t.Errorf("Expected the company day on 01/04/2019 to start midnight of the provided date\nGot %s to %s", span.Start(), span.End())
			}
			testDate, _ = time.ParseInLocation(timeFormat, "2019-04-02T00:00:00", aklTz)
			if !span.End().Equal(testDate) {
				t.Errorf("Expected the company day on 01/04/2019 to end midnight next day\nGot %s to %s", span.Start(), span.End())
			}
		default:
			t.Errorf("company day should have started on a , got %s", span.Start().Weekday())
			fmt.Printf("%s - %s\n", span.Start(), span.End())
		}
	}
}

func TestCalendarTimezones(t *testing.T) {
	var err error
	start, err := time.ParseInLocation(timeFormat, firstJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}
	end, err := time.ParseInLocation(timeFormat, lastJan, time.UTC)
	if err != nil {
		t.Errorf("Time parse error: %s", err.Error())
	}

	config.GlobalConfig = config.Config{
		Holidays:     []string{"Auckland Anniversary", "Wellington Anniversary"},
		CalendarURL:  "http://apps.employment.govt.nz/ical/public-holidays-all.ics",
		Timezone:     "America/New_York",
		ScheduleSpan: timespan.New(start, end),
	}

	cal := NewCalendarDataSource()

	for _, span := range cal.Spans() {
		day := span.Start()
		spanTz := time.Date(day.Year(), day.Month(), day.Day(), day.Hour(), day.Minute(), day.Second(), day.Nanosecond(), day.Location())
		testTz := time.Date(day.Year(), day.Month(), day.Day(), day.Hour(), day.Minute(), day.Second(), day.Nanosecond(), nyTz)

		if !spanTz.Equal(testTz) {
			t.Errorf("Expected timezone %s, got %s", testTz, spanTz)
		}
	}
}
