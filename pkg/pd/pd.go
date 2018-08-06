package pd

import (
	"time"

	"github.com/leosunmo/pagerduty-schedule/pkg/calendar"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/leosunmo/pagerduty-schedule/pkg/config"
)

const pdTimeFormat = "2006-01-02T15:04:05-07:00"

// NewPDClient returns a PagerDuty client created using the provided auth token
func NewPDClient(authtoken string) *pagerduty.Client {
	return pagerduty.NewClient(authtoken)
}

// ReadShifts parses the Pagerduty shifts and tags each hour in the shift with
// an hour-type, business hour, afterhours, stat holiday etc.
func ReadShifts(client *pagerduty.Client, conf *config.ScheduleConfig, schedule string, startDate, endDate time.Time) (UserShifts, error) {
	getschopts := pagerduty.GetScheduleOptions{
		Since: startDate.String(),
		Until: endDate.String(),
	}
	us := make(UserShifts)
	if ds, err := client.GetSchedule(schedule, getschopts); err != nil {
		panic(err)
	} else {
		for _, se := range ds.FinalSchedule.RenderedScheduleEntries {
			startTime, terr := time.Parse(pdTimeFormat, se.Start)
			if terr != nil {
				return nil, terr
			}
			endTime, terr := time.Parse(pdTimeFormat, se.End)
			if terr != nil {
				return nil, terr
			}
			s := Shift{
				StartDate:  startTime,
				EndDate:    endTime,
				Duration:   endTime.Sub(startTime),
				ShiftHours: make(map[time.Time]int),
				Calendar:   calendar.NewCalendar(startDate, endDate, conf),
			}
			s.ProcessHours()

			us[se.User.Summary] = append(us[se.User.Summary], s)
		}
	}
	return us, nil
}
