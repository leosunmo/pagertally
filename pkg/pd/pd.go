package pd

import (
	"time"

	"github.com/leosunmo/pagerduty-shifts/pkg/calendar"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/leosunmo/pagerduty-shifts/pkg/config"
)

const pdTimeFormat = "2006-01-02T15:04:05-07:00"

// NewPDClient returns a PagerDuty client created using the provided auth token
func NewPDClient(authtoken string) *pagerduty.Client {
	return pagerduty.NewClient(authtoken)
}

// ReadShifts parses the Pagerduty shifts and tags each hour in the shift with
// an hour-type, business hour, afterhours, stat holiday etc.
func ReadShifts(client *pagerduty.Client, conf *config.ScheduleConfig, cal *calendar.Calendar, schedule string, startDate, endDate time.Time) (string, UserShifts, error) {
	getschopts := pagerduty.GetScheduleOptions{
		Since: startDate.String(),
		Until: endDate.String(),
	}
	var scheduleName string
	us := make(UserShifts)
	ds, err := client.GetSchedule(schedule, getschopts)
	if err != nil {
		return "", nil, err
	}
	scheduleName = ds.Name
	for _, se := range ds.FinalSchedule.RenderedScheduleEntries {
		startTime, terr := time.Parse(pdTimeFormat, se.Start)
		if terr != nil {
			return "", nil, terr
		}
		endTime, terr := time.Parse(pdTimeFormat, se.End)
		if terr != nil {
			return "", nil, terr
		}
		s := Shift{
			StartDate:    startTime,
			EndDate:      endTime,
			Duration:     endTime.Sub(startTime),
			ScheduleName: ds.Name,
			ShiftHours:   make(map[time.Time]int),
			Calendar:     cal,
		}
		s.ProcessHours()

		us[se.User.Summary] = append(us[se.User.Summary], s)
	}

	return scheduleName, us, nil
}
