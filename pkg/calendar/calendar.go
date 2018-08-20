package calendar

import (
	"fmt"
	"strings"
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
	calTimezone    *time.Location
}

// NewCalendar returns an empty calendar
func NewCalendar(startDate, endDate time.Time, conf *config.ScheduleConfig) *Calendar {

	// Get a slice of all days between the start and end dates of the schedule
	calDays := []time.Time{}
	startDate = FlattenTime(startDate)
	endDate = FlattenTime(endDate)
	tr := timerange.New(startDate, endDate, time.Hour*24)
	for tr.Next() {
		calDays = append(calDays, tr.Current())
	}
	loc, err := time.LoadLocation(conf.Timezone)

	if err != nil {
		panic("Failed loading location from timezone provided")
	}
	// Get the calendar timezone in second offsets

	cal := Calendar{
		calStart:       startDate,
		calEnd:         endDate,
		calDays:        calDays,
		calendarHours:  make(map[time.Time]int, 0),
		scheduleConfig: conf,
		calTimezone:    loc,
	}
	cal.tagAfterhoursAndWeekends()
	err = cal.parseAndFilterPublicHolidayiCal(cal.scheduleConfig.CalendarURL)
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
			schedDay = FlattenDate(schedDay)
			events, exists := eventsByDates[schedDay.Format(YmdHis)]
			if !exists {
				continue
			}
			for _, event := range events {
				// See if event is in event whitelist
				if c.filterEvent(event.GetSummary()) {
					// Start iterating over every hour of the event and add those hours as stat days
					tr := timerange.New(event.GetStart(), event.GetEnd().Add(time.Duration(-1)*time.Hour), time.Hour)
					for tr.Next() {
						adjustedTime := AdjustForTimezone(tr.Current(), c.scheduleConfig.ParsedTimezone)
						c.addHour(adjustedTime, StatHolidayHour)
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
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			tr := timerange.New(day, day.Add(time.Hour*24), time.Hour)
			for tr.Next() {
				if c.calendarHours[tr.Current()] != StatHolidayHour {
					c.addHour(tr.Current(), WeekendHour)
				}
			}
			continue
		}
		// Add afterhours from start of day (00:01) to start of business hours (eg. 09:00)
		tr := timerange.New(day, day.Add(time.Hour*time.Duration(bStart.Hour())), time.Hour)
		for tr.Next() {
			if c.calendarHours[tr.Current()] != StatHolidayHour {
				c.addHour(tr.Current(), BusinessAfterHour)
			}
		}
		// Add afterhours from business hours end (eg. 17:00) to end of day (day + 23 hours to avoid adding an extra hour at the end of the day)
		// unless it's Friday, then it's weekend hours.
		if day.Weekday() != time.Friday {
			tr = timerange.New(day.Add(time.Hour*time.Duration(bEnd.Hour())), day.Add(time.Hour*23), time.Hour)
			for tr.Next() {
				if c.calendarHours[tr.Current()] != StatHolidayHour {
					c.addHour(tr.Current(), BusinessAfterHour)
				}
			}
		} else {
			tr := timerange.New(day.Add(time.Hour*time.Duration(bEnd.Hour())), day.Add(time.Hour*24), time.Hour)
			for tr.Next() {
				if c.calendarHours[tr.Current()] != StatHolidayHour {
					c.addHour(tr.Current(), WeekendHour)
				}
			}
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
		n := time.Date(2006, 01, 02, hour, min, 0, 0, timestamp.Location())
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

// AdjustForTimezone takes a timestamp and adds the offset of the
// provided timezone location and then returns the timestamp
// with the offset added/removed presented in the correct timezone
func AdjustForTimezone(t time.Time, loc *time.Location) time.Time {
	_, tzOffsetSeconds := t.In(loc).Zone()
	return t.Add(time.Second * time.Duration(-tzOffsetSeconds)).In(loc)
}

// SheetDurationFormat formats the default Duration.String() string to
// Google sheets timeformat. Nanoseconds ignored.
// 48h30m25s -> 48:30:25.000
func SheetDurationFormat(d time.Duration) string {
	ds := d.String()
	h := strings.Split(ds, "h")
	m := strings.Split(h[1], "m")
	s := strings.Split(m[1], "s")
	return fmt.Sprintf("%s:%s:%s.000", h[0], m[0], s[0])
}
