package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/leosunmo/pagerduty-schedule/pkg/calendar"

	"github.com/leosunmo/pagerduty-schedule/pkg/config"
	"github.com/leosunmo/pagerduty-schedule/pkg/pd"
)

func main() {
	var authtoken string
	var schedule string
	var configPath string
	flag.StringVar(&authtoken, "token", "", "Provide PagerDuty API token")
	flag.StringVar(&schedule, "schedule", "", "Provide PagerDuty schedule ID")
	flag.StringVar(&configPath, "conf", "", "Provide config file path")

	flag.Parse()
	if authtoken == "" {
		fmt.Println("Please provide PagerDuty API token")
		flag.Usage()
		os.Exit(1)
	}
	if schedule == "" {
		fmt.Println("Please provide PagerDuty schedule ID")
		flag.Usage()
		os.Exit(1)
	}
	if configPath == "" {
		fmt.Println("Please provide a config file")
		flag.Usage()
		os.Exit(1)
	}

	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	pdClient := pd.NewPDClient(authtoken)
	conf := config.GetScheduleConfig(configPath)
	userShifts, err := pd.ReadShifts(pdClient, conf, schedule, startDate, endDate)
	if err != nil {
		panic(err)
	}
	fmt.Printf("UserShifts: %+v", userShifts)
	for user, shifts := range userShifts {
		fmt.Printf("\nUser: %s\n", user)
		fmt.Println("Shifts:")
		for i, shift := range shifts {
			fmt.Printf("\nShift %d:\n", i)
			fmt.Printf("Start: %s\nEnd: %s\n", shift.StartDate, shift.EndDate)
			fmt.Printf("Duration: %s\n", shift.Duration)
			var bh, bah, wh, sh int
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
			fmt.Printf("BusinessHours: %d\tAfterHours: %d\nWeekendHours: %d\tStatDaysHours: %d\n", bh, bah, wh, sh)
		}
	}

	// totalShifts := make(map[string]time.Duration)
	// for user, shifts := range userShifts {
	// 	totalShifts[us.] = totalShifts[us.username] + us.shiftDur
	// }
	// for user, totalDur := range totalShifts {
	// 	fmt.Printf("User: %s\nTotal on-call: %s\n", user, totalDur)
	// }
}
