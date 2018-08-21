package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/leosunmo/pagerduty-schedule/pkg/calendar"
	"github.com/leosunmo/pagerduty-schedule/pkg/config"
	"github.com/leosunmo/pagerduty-schedule/pkg/outputs"
	"github.com/leosunmo/pagerduty-schedule/pkg/pd"
)

type schedulesListFlag []string

func (s *schedulesListFlag) String() string {
	return fmt.Sprint(*s)
}

func (s *schedulesListFlag) Set(value string) error {
	if len(*s) > 0 {
		return errors.New("schedules flag already set")
	}
	for _, scheduleID := range strings.Split(value, ",") {
		*s = append(*s, scheduleID)
	}
	return nil
}

type finalShifts map[string]finalOutput

type finalOutput struct {
	BusinessHours int
	AfterHours    int
	WeekendHours  int
	StatHours     int
	TotalHours    int
	TotalShifts   int
	TotalDuration time.Duration
}

func main() {
	var authtoken string
	var schedules schedulesListFlag
	var configPath string
	var outputFile string
	var gsheetid string
	var startMonth string
	var timeZone string
	flag.StringVar(&authtoken, "token", "", "Provide PagerDuty API token")
	flag.Var(&schedules, "schedules", "Comma separated list of PagerDuty schedule IDs")
	flag.StringVar(&configPath, "conf", "", "Provide config file path")
	flag.StringVar(&outputFile, "outfile", "", "(Optional) Print as CSV to this file")
	flag.StringVar(&gsheetid, "gsheetid", "", "(Optional) Print to Google Sheet ID provided")
	flag.StringVar(&startMonth, "month", "", "(Optional) Provide the month you want to process. Default current month")
	flag.StringVar(&timeZone, "timezone", "", "(Optional) Force timezone. Defaults to local")

	flag.Parse()
	if authtoken == "" {
		fmt.Println("Please provide PagerDuty API token")
		flag.Usage()
		os.Exit(1)
	}
	if len(schedules) < 1 {
		fmt.Println("Please provide at least one PagerDuty schedule ID")
		flag.Usage()
		os.Exit(1)
	}
	if configPath == "" {
		fmt.Println("Please provide a config file")
		flag.Usage()
		os.Exit(1)
	}

	var startDate time.Time
	var err error
	// If timezone isn't set, default to the local location
	if timeZone == "" {
		timeZone = time.Local.String()
	}
	// Create a time.Location using the timeZone that we can use for parsing
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		log.Fatalf("Failed to parse timezone. use IANA TZ format, err: %s", err.Error())
	}

	if startMonth != "" {
		startDate, err = time.ParseInLocation("January 2006", fmt.Sprintf("%s %d", startMonth, time.Now().Year()), loc)
		if err != nil {
			log.Fatalf("Unable to parse month, err: %s\n", err.Error())
		}
	} else {
		startDate, err = time.ParseInLocation("January 2006", fmt.Sprintf("%s %d", time.Now().Month(), time.Now().Year()), loc)
		if err != nil {
			log.Fatalf("Unable to parse month, err: %s\n", err.Error())
		}
	}
	endDate := startDate.AddDate(0, +1, 0)
	conf := config.GetScheduleConfig(configPath)
	pdClient := pd.NewPDClient(authtoken)
	cal := calendar.NewCalendar(startDate, endDate, conf)
	totalUserShifts := pd.ScheduleUserShifts{}

	for _, schedule := range schedules {
		scheduleName, userShifts, err := pd.ReadShifts(pdClient, conf, cal, schedule, startDate, endDate)
		if err != nil {
			log.Fatal(err.Error())
		}
		totalUserShifts[scheduleName] = userShifts
	}

	// Let's count up the number of hours for each person, adding up all their shifts
	fo := outputs.CalculateFinalOutput(totalUserShifts)

	if outputFile == "" && gsheetid == "" {
		var scheduleNames []string
		for sNames := range totalUserShifts {
			scheduleNames = append(scheduleNames, sNames)
		}
		fmt.Printf("Schedules: %s", strings.Join(scheduleNames, " & "))
		for user, o := range fo {
			fmt.Printf("\nUser: %s\n", user)
			fmt.Printf("BusinessHours: %d\tAfterHours: %d\nWeekendHours: %d\tStatDaysHours: %d\n"+
				"\nTotal Hours: %d\tTotal Shifts: %d\nTotal Duration on-call: %s\n",
				o.BusinessHours, o.AfterHours, o.WeekendHours,
				o.StatHours, o.TotalHours, o.TotalShifts, o.TotalDuration.String())
		}
	} else if outputFile != "" {

		o := outputs.NewCSVOutput(outputFile)
		err := outputs.PrintOutput(o, fo)
		if err != nil {
			log.Fatal(err)
		}

	} else if gsheetid != "" {
		o := outputs.NewGSheetOutput(gsheetid, startMonth+" "+strconv.Itoa(time.Now().Year()), "A1", "service-account-dev.json")
		err := outputs.PrintOutput(o, fo)
		if err != nil {
			log.Fatal(err)
		}
	}

}
