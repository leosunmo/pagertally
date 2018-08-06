package calendar

import (
	"fmt"
	"time"

	timerange "github.com/leosunmo/timerange-go"

	ics "github.com/leosunmo/ics-golang"
	"github.com/leosunmo/pagerduty-schedule/pkg/config"
)

// BusinessHour is an hour marked as within business hours, 9:00 to 18:00
const BusinessHour int = 1

// BusinessAfterHour is an hour after normal business hours, 18:00 to 9:00
const BusinessAfterHour int = 2

// WeekendHour is an hour within weekends, 18:00 Friday until 9:00 Monday
const WeekendHour int = 3

// StatHolidayHour is an hour within statutory holidays, 00:00 to 00:00
const StatHolidayHour int = 4

// YmdHis is the timeformat the iCal parser expects for event dates
const YmdHis string = "2006-01-02 15:04:05"

// Calendar containsh all hours of national and
// regional holidays (that we whitelisted) as well as
// the configuration of whitelisted holidays and
// business hours
type Calendar struct {
	calStart       time.Time
	calEnd         time.Time
	calDays        []time.Time
	calendarHours  map[time.Time]int
	scheduleConfig *config.ScheduleConfig
}

// NewCalendar returns an empty calendar
func NewCalendar(startDate, endDate time.Time, conf *config.ScheduleConfig) *Calendar {
	startDate = startDate.UTC()
	endDate = endDate.UTC()

	// Get a slice of all days between the start and end dates of the schedule
	calDays := []time.Time{}
	startDate = FlattenTime(startDate)
	endDate = FlattenTime(endDate)
	tr := timerange.New(startDate, endDate, time.Hour*24)
	for tr.Next() {
		calDays = append(calDays, tr.Current())
	}
	cal := Calendar{
		calStart:       startDate,
		calEnd:         endDate,
		calDays:        calDays,
		calendarHours:  make(map[time.Time]int, 0),
		scheduleConfig: conf,
	}
	cal.tagAfterhoursAndWeekends()
	err := cal.parseAndFilterPublicHolidayiCal(cal.scheduleConfig.CalendarURL)
	if err != nil {
		panic(err)
	}
	return &cal
}

func (c *Calendar) GetBusinessHours() (time.Time, time.Time) {
	return c.scheduleConfig.GetBusinessHours()
}
func (c *Calendar) addHour(hourStart time.Time, hourType int) {
	c.calendarHours[hourStart] = hourType
}

func (c *Calendar) parseAndFilterPublicHolidayiCal(icsLink string) error {
	//  create new parser
	parser := ics.New()

	// get the input chan
	inputChan := parser.GetInputChan()

	// send the calendar urls to be parsed
	//inputChan <- "http://apps.employment.govt.nz/ical/public-holidays-all.ics"
	inputChan <- icsLink
	//  wait for the calendar to be parsed
	parser.Wait()

	// get all calendars in this parser
	cals, err := parser.GetCalendars()
	if err != nil {
		return fmt.Errorf("Failed to parse iCal")
	}
	for _, cal := range cals {
		eventsByDates := cal.GetEventsByDates()
		for _, schedDay := range c.calDays {
			events, exists := eventsByDates[schedDay.Format(YmdHis)]
			if !exists {
				continue
			}
			for _, event := range events {
				// See if event is in event whitelist
				if c.filterEvent(event.GetSummary()) {
					// Start iterating over every hour of the event and add those hours as stat days
					// Convert to UTC
					tr := timerange.New(event.GetStart().UTC(), event.GetEnd().UTC(), time.Hour)
					for tr.Next() {
						c.addHour(tr.Current(), StatHolidayHour)
					}
				}
			}
		}
	}
	return nil
}

// filterEvent compares the given event name against the whitelist of events
// specified in the config.
// returns true if it's whitelisted, false if it should be ignored
func (c *Calendar) filterEvent(eventName string) bool {
	for _, h := range c.scheduleConfig.Holidays {
		if eventName == h {
			return true
		}
	}
	return false
}

func (c *Calendar) tagAfterhoursAndWeekends() {
	bStart, bEnd := c.GetBusinessHours()
	for _, day := range c.calDays {
		if day.Weekday() == time.Friday {
			tr := timerange.New(day.Add(time.Hour*time.Duration(bEnd.Hour())), day.Add(time.Hour*24), time.Hour)
			for tr.Next() {
				c.addHour(tr.Current(), WeekendHour)
			}
			continue
		}
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			tr := timerange.New(day, day.Add(time.Hour*24), time.Hour)
			for tr.Next() {
				c.addHour(tr.Current(), WeekendHour)
			}
			continue
		}
		tr := timerange.New(day, day.Add(time.Hour*time.Duration(bStart.Hour())), time.Hour)
		for tr.Next() {
			c.addHour(tr.Current(), BusinessAfterHour)
		}
		tr = timerange.New(day.Add(time.Hour*time.Duration(bEnd.Hour())), day.Add(time.Hour*23), time.Hour)
		for tr.Next() {
			c.addHour(tr.Current(), BusinessAfterHour)
		}
	}
}

// GetHourTag returns the hour type of the timestamp provided
func (c *Calendar) GetHourTag(h time.Time) int {
	hourType, exists := c.calendarHours[h]
	if !exists {
		return BusinessHour
	}
	return hourType

}

func timeWithinTimeRange(start time.Time, end time.Time, timestamp time.Time) bool {
	// normalise the date part
	timeMap := make(map[string]time.Time)
	timeMap["start"] = start
	timeMap["end"] = end
	timeMap["timestamp"] = timestamp
	for k, v := range timeMap {
		hour, min, _ := v.Clock()
		n := time.Date(2006, 01, 02, hour, min, 0, 0, time.UTC)
		timeMap[k] = n
	}
	if timeMap["timestamp"].After(timeMap["start"]) && timeMap["timestamp"].Before(timeMap["emd"]) {
		return true
	}
	return false
}

func dateWithinDateRange(start time.Time, end time.Time, datestamp time.Time) bool {
	if datestamp.After(start) && datestamp.Before(end) {
		return true
	}
	return false
}

// FlattenDate returns the timestamp without hours, mins or seconds
// this is because time.Time.Truncate() doesn't work with non-UTC time
func FlattenDate(t time.Time) time.Time {
	y, m, d := t.Date()
	loc := t.Location()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// FlattenTime returns the timestamp without minutes or seconds
// this is because time.Time.Truncate() doesn't work with non-UTC time
func FlattenTime(t time.Time) time.Time {
	y, m, d := t.Date()
	h, _, _ := t.Clock()
	loc := t.Location()
	return time.Date(y, m, d, h, 0, 0, 0, loc)
}