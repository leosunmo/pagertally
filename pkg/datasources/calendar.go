package datasources

// iCal calendar parser (for public holidays off NZ public holidays source)

import (
	"fmt"
	"time"

	ics "github.com/leosunmo/ics-golang"
	"github.com/leosunmo/pagertally/pkg/config"
	"github.com/leosunmo/pagertally/pkg/timespan"
	log "github.com/sirupsen/logrus"
)

// YmdHis is the timeformat the iCal parser expects for event dates
const YmdHis string = "2006-01-02 15:04:05"

// CalendarDataSource is an iCal Datasource that has lists of timespans
// based on a whitelist of events provided in configuration
type CalendarDataSource struct {
	CalTotalSpan  timespan.Span
	CalendarSpans []timespan.Span
	CalTimezone   *time.Location
}

// NewCalendarDataSource returns a DataSource populated by events provided in configured iCal
func NewCalendarDataSource() CalendarDataSource {
	startDate := config.GlobalConfig.ScheduleSpan.Start()
	endDate := config.GlobalConfig.ScheduleSpan.End()
	loc := config.Timezone()

	cal := CalendarDataSource{
		CalTotalSpan: timespan.New(startDate, endDate),
		CalTimezone:  loc,
	}
	err := cal.parseAndFilterPublicHolidayiCal()
	if err != nil {
		log.Fatalf("datasources/calendar: failed to retrieve public holidays, err: %s", err.Error())
		return cal
	}

	return cal
}

// Spans returns the timespans from the CalendarSource
func (c CalendarDataSource) Spans() []timespan.Span {
	return c.CalendarSpans
}

func (c *CalendarDataSource) parseAndFilterPublicHolidayiCal() error {
	//  create new parser
	parser := ics.New()
	parser.DefaultTimezone(c.CalTimezone)

	// get the input chan
	inputChan := parser.GetInputChan()

	// send the calendar urls to be parsed
	//inputChan <- "http://apps.employment.govt.nz/ical/public-holidays-all.ics"
	inputChan <- config.GlobalConfig.CalendarURL
	//  wait for the calendar to be parsed
	parser.Wait()

	parseErrors, _ := parser.GetErrors()
	if len(parseErrors) > 0 {
		if len(parseErrors) == 1 {
			return parseErrors[0]
		}
		for _, err := range parseErrors {
			log.Errorf("\t%s", err.Error())
		}
		return fmt.Errorf("multiple calendar parse errors")
	}

	// get all calendars in this parser
	cals, err := parser.GetCalendars()
	if err != nil {
		return fmt.Errorf("Failed to parse iCal, err: %s", err.Error())
	}

	for _, cal := range cals {
		tzCal := cal.SetTimezone(*c.CalTimezone)
		for _, schedDay := range c.CalTotalSpan.Dates() {
			events, found := tzCal.GetEventsByDate(schedDay)
			if !found {
				continue
			}
			for _, event := range events {
				// See if event is in event whitelist
				if c.filterEvent(event.GetSummary()) {
					span := timespan.New(event.GetStart(), event.GetEnd())
					c.addSpan(span)
				}
			}
		}
	}
	return nil
}

// filterEvent compares the given event name against the whitelist of events
// specified in the config.
// returns true if it's whitelisted, false if it should be ignored
func (c *CalendarDataSource) filterEvent(eventName string) bool {
	for _, h := range config.GlobalConfig.Holidays {
		if eventName == h {
			return true
		}
	}
	return false
}

// addSpan adds the span to the calendar slice of spans
//
// If the new span overlaps with any existing span in the calendar
// slice of spans we trim it so there's no overlap
func (c *CalendarDataSource) addSpan(span timespan.Span) {
	for _, existingSpan := range c.CalendarSpans {
		if trimmedSpan, overlap := span.TrimIfOverlaps(existingSpan); overlap {
			if trimmedSpan.IsZero() {
				return
			}
			span = trimmedSpan
		}
	}
	c.CalendarSpans = append(c.CalendarSpans, span)
}
