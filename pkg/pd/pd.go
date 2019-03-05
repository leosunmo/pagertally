package pd

import (
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/leosunmo/pagertally/pkg/timespan"
)

const pdTimeFormat = "2006-01-02T15:04:05-07:00"

// NewPDClient returns a PagerDuty client created using the provided auth token
func NewPDClient(authtoken string) *pagerduty.Client {
	return pagerduty.NewClient(authtoken)
}

// ReadShifts returns a UserShift per schedule in a ScheduleUserShifts map from PagerDuty
func ReadShifts(client *pagerduty.Client, PdSchedules []string, startDate, endDate time.Time) (timespan.ScheduleUserShifts, error) {
	getschopts := pagerduty.GetScheduleOptions{
		Since: startDate.String(),
		Until: endDate.String(),
	}
	schdUserShifts := make(timespan.ScheduleUserShifts)
	for _, PdSchedule := range PdSchedules {
		us := make(timespan.UserShifts)
		ds, err := client.GetSchedule(PdSchedule, getschopts)
		if err != nil {
			return nil, err
		}
		for _, se := range ds.FinalSchedule.RenderedScheduleEntries {
			startTime, terr := time.Parse(pdTimeFormat, se.Start)
			if terr != nil {
				return nil, terr
			}
			endTime, terr := time.Parse(pdTimeFormat, se.End)
			if terr != nil {
				return nil, terr
			}
			shiftSpan := timespan.New(startTime, endTime)
			user := timespan.User{
				Name:     se.User.Summary,
				Location: startTime.Location(),
			}
			us[user] = append(us[user], shiftSpan)
		}
		schdUserShifts[timespan.ScheduleName(ds.Name)] = us
	}

	return schdUserShifts, nil
}
