package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

type userSchedule struct {
	username string

	shiftStart time.Time
	shiftEnd   time.Time
	shiftDur   time.Duration
}

const pgTimeFormat = "2006-01-02T15:04:05-07:00"

func main() {
	var authtoken string
	var schedule string
	flag.StringVar(&authtoken, "token", "", "Provide PagerDuty API token")
	flag.StringVar(&schedule, "schedule", "", "Provide PagerDuty schedule ID")
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
	client := pagerduty.NewClient(authtoken)

	getschopts := pagerduty.GetScheduleOptions{
		Since: time.Now().AddDate(0, -1, 0).String(),
		Until: time.Now().String(),
	}
	userShifts := make([]userSchedule, 0)
	if ds, err := client.GetSchedule(schedule, getschopts); err != nil {
		panic(err)
	} else {
		fmt.Println(ds.Name)
		fmt.Println(ds.HTMLURL)
		for _, se := range ds.FinalSchedule.RenderedScheduleEntries {
			startTime, _ := time.Parse(pgTimeFormat, se.Start)
			endTime, _ := time.Parse(pgTimeFormat, se.End)

			userShifts = append(userShifts, userSchedule{
				username:   se.User.Summary,
				shiftStart: startTime,
				shiftEnd:   endTime,
				shiftDur:   endTime.Sub(startTime),
			})
		}
	}

	totalShifts := make(map[string]time.Duration)
	for _, us := range userShifts {
		totalShifts[us.username] = totalShifts[us.username] + us.shiftDur
	}
	for user, totalDur := range totalShifts {
		fmt.Printf("User: %s\nTotal on-call: %s\n", user, totalDur)
	}
}
