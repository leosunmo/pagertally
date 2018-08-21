package outputs

import (
	"time"

	"github.com/leosunmo/pagerduty-schedule/pkg/calendar"
	"github.com/leosunmo/pagerduty-schedule/pkg/pd"
)

func CalculateFinalOutput(totalUserShifts pd.ScheduleUserShifts) FinalShifts {
	fo := make(FinalShifts, 0)
	for _, userShifts := range totalUserShifts {
		for user, shifts := range userShifts {
			var bh, bah, wh, sh, ts int
			var td time.Duration
			for _, shift := range shifts {
				td = td + shift.Duration
				for _, t := range shift.ShiftHours {
					switch t {
					case calendar.BusinessHour:
						bh++
					case calendar.BusinessAfterHour:
						bah++
					case calendar.WeekendHour:
						wh++
					case calendar.StatHolidayHour:
						sh++
					}
				}
				// Count number of shifts
				ts++
			}
			// Add it all to a map of output struct
			if _, exists := fo[user]; !exists {
				fo[user] = finalOutput{
					TotalShifts:   ts,
					BusinessHours: bh,
					AfterHours:    bah,
					WeekendHours:  wh,
					StatHours:     sh,
					TotalHours:    bh + bah + wh + sh,
					TotalDuration: td,
				}
			} else {
				tfo := fo[user]
				tfo.TotalShifts += ts
				tfo.BusinessHours += bh
				tfo.AfterHours += bah
				tfo.WeekendHours += wh
				tfo.StatHours += sh
				tfo.TotalHours += bh + bah + wh + sh
				tfo.TotalDuration += td
				fo[user] = tfo
			}
		}
	}
	return fo
}

func PrintOutput(o Output, fs FinalShifts) error {
	data := make([][]interface{}, 0)
	for u, fo := range fs {
		row := []interface{}{u, fo.BusinessHours, fo.AfterHours, fo.WeekendHours,
			fo.StatHours, fo.TotalHours, fo.TotalShifts, calendar.SheetDurationFormat(fo.TotalDuration)}
		data = append(data, row)
	}
	o.Print(data)
	return nil
}
