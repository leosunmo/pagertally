package datasources

import (
	"time"

	"github.com/leosunmo/pagertally/pkg/config"
	"github.com/leosunmo/pagertally/pkg/timespan"
	timerange "github.com/leosunmo/timerange-go"
)

// WeekendDataSource is a datasource for generic after-hours and weekend
// spans.
type WeekendDataSource struct {
	WeekendSpans []timespan.Span
}

// AfterHoursDataSource is a datasource for generic after-hours and weekend
// spans.
type AfterHoursDataSource struct {
	AfterHoursSpans []timespan.Span
}

// NewWeekendDataSource returns a DataSource with weekend spans
func NewWeekendDataSource() WeekendDataSource {

	wds := WeekendDataSource{
		WeekendSpans: []timespan.Span{},
	}
	// Create and iterate of a timerange of the entire (usually month-long) schedule that we are processing
	tr := timerange.New(config.GlobalConfig.ScheduleSpan.Start(), config.GlobalConfig.ScheduleSpan.End(), time.Hour*24)
	for tr.Next() {
		wds.attributeWeekends(tr.Current())
	}
	wds.WeekendSpans = timespan.MergeSpans(wds.WeekendSpans)

	return wds
}

// NewAfterHoursDataSource returns a DataSource with afterhours spans
func NewAfterHoursDataSource() AfterHoursDataSource {
	ahds := AfterHoursDataSource{
		AfterHoursSpans: []timespan.Span{},
	}
	// Create and iterate of a timerange of the entire (usually month-long) schedule that we are processing
	tr := timerange.New(config.GlobalConfig.ScheduleSpan.Start(), config.GlobalConfig.ScheduleSpan.End(), time.Hour*24)
	for tr.Next() {
		ahds.attributeAfterHours(tr.Current())
	}
	ahds.AfterHoursSpans = timespan.MergeSpans(ahds.Spans())
	return ahds
}

// Spans returns weekend attributed spans
func (wds WeekendDataSource) Spans() []timespan.Span {
	return wds.WeekendSpans
}

// Spans returns out of business hours attributed spans
func (ahds AfterHoursDataSource) Spans() []timespan.Span {
	return ahds.AfterHoursSpans
}

func (wds *WeekendDataSource) attributeWeekends(day time.Time) {

	// Get configured business open hours
	bStart, bEnd := config.BusinessHoursForDate(day)

	switch day.Weekday() {
	case time.Saturday, time.Sunday:
		// Add 24 hour weekend spans for Saturday and Sunday
		startTime := day
		//endTime := timespan.EndOfDay(day)
		endTime := day.AddDate(0, 0, 1)
		span := timespan.New(startTime, endTime)
		wds.WeekendSpans = append(wds.WeekendSpans, span)
	case time.Friday:
		// If it's Friday we add a weekend span from close of business (COB) to 23:59
		startTime := time.Date(day.Year(), day.Month(), day.Day(), bEnd.Hour(), bEnd.Minute(), bEnd.Second(), bEnd.Nanosecond(), day.Location())
		//endTime := timespan.EndOfDay(day)
		endTime := day.AddDate(0, 0, 1)
		span := timespan.New(startTime, endTime)

		wds.WeekendSpans = append(wds.WeekendSpans, span)
	case time.Monday:
		// If it's Monday we add a weekend span from 00:00 to opening of business (OOB)
		startTime := day
		endTime := time.Date(day.Year(), day.Month(), day.Day(), bStart.Hour(), bStart.Minute(), bStart.Second(), bStart.Nanosecond(), day.Location())
		span := timespan.New(startTime, endTime)
		wds.WeekendSpans = append(wds.WeekendSpans, span)
	}
}

func (ahds *AfterHoursDataSource) attributeAfterHours(day time.Time) {
	// Get configured business open hours
	bStart, bEnd := config.BusinessHoursForDate(day)

	switch day.Weekday() {
	case time.Friday:
		// If it's Friday, add only a monrning AfterHours span since the time after COB is weekend.
		startTime := day
		endTime := time.Date(day.Year(), day.Month(), day.Day(), bStart.Hour(), bStart.Minute(), bStart.Second(), bStart.Nanosecond(), day.Location())
		span := timespan.New(startTime, endTime)
		ahds.AfterHoursSpans = append(ahds.AfterHoursSpans, span)
	case time.Monday:
		// If it's Monday, add only an evening AfterHours span since the time before OOB is weekend.
		startTime := time.Date(day.Year(), day.Month(), day.Day(), bEnd.Hour(), bEnd.Minute(), bEnd.Second(), bEnd.Nanosecond(), day.Location())
		//endTime := timespan.EndOfDay(day)
		endTime := day.AddDate(0, 0, 1)
		span := timespan.New(startTime, endTime)
		ahds.AfterHoursSpans = append(ahds.AfterHoursSpans, span)
	case time.Tuesday, time.Wednesday, time.Thursday:
		// If it's not Sunday or Saturday (and not Friday since we caught that above) we add morning and evening AfterHours spans

		// Morning span
		startTime := day
		endTime := time.Date(day.Year(), day.Month(), day.Day(), bStart.Hour(), bStart.Minute(), bStart.Second(), bStart.Nanosecond(), day.Location())
		span := timespan.New(startTime, endTime)
		ahds.AfterHoursSpans = append(ahds.AfterHoursSpans, span)

		// Afternoon span
		startTime = time.Date(day.Year(), day.Month(), day.Day(), bEnd.Hour(), bEnd.Minute(), bEnd.Second(), bEnd.Nanosecond(), day.Location())
		//endTime = timespan.EndOfDay(day)
		endTime = day.AddDate(0, 0, 1)
		span = timespan.New(startTime, endTime)
		ahds.AfterHoursSpans = append(ahds.AfterHoursSpans, span)

	}
}
