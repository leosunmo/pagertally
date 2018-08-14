package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/leosunmo/pagerduty-schedule/pkg/calendar"

	"github.com/leosunmo/pagerduty-schedule/pkg/config"
	"github.com/leosunmo/pagerduty-schedule/pkg/pd"
)

type finalShifts map[string]finalOutput

type finalOutput struct {
	businessHours int
	afterHours    int
	weekendHours  int
	statHours     int
	totalHours    int
	totalShifts   int
	totalDuration time.Duration
}

func main() {
	var authtoken string
	var schedule string
	var configPath string
	var outputFile string
	flag.StringVar(&authtoken, "token", "", "Provide PagerDuty API token")
	flag.StringVar(&schedule, "schedule", "", "Provide PagerDuty schedule ID")
	flag.StringVar(&configPath, "conf", "", "Provide config file path")
	flag.StringVar(&outputFile, "outfile", "", "(Optional) Print as CSV to this file")

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

	// startDate, err := time.Parse(time.RFC3339, "2018-04-01T00:00:00+12:00")
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }
	startDate := time.Now()
	endDate := startDate.AddDate(0, +1, 0)

	pdClient := pd.NewPDClient(authtoken)
	conf := config.GetScheduleConfig(configPath)
	cal := calendar.NewCalendar(startDate, endDate, conf)
	userShifts, err := pd.ReadShifts(pdClient, conf, cal, schedule, startDate, endDate)
	if err != nil {
		panic(err)
	}
	// Let's count up the number of hours for each person, adding up all their shifts
	fo := make(finalShifts, 0)
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
		fo[user] = finalOutput{
			totalShifts:   ts,
			businessHours: bh,
			afterHours:    bah,
			weekendHours:  wh,
			statHours:     sh,
			totalHours:    bh + bah + wh + sh,
			totalDuration: td,
		}
	}
	if outputFile == "" {
		for user, o := range fo {
			fmt.Printf("\nUser: %s\n", user)
			fmt.Printf("BusinessHours: %d\tAfterHours: %d\nWeekendHours: %d\tStatDaysHours: %d\n"+
				"\nTotal Hours: %d\tTotal Shifts: %d\nTotal Duration on-call: %s\n",
				o.businessHours, o.afterHours, o.weekendHours,
				o.statHours, o.totalHours, o.totalShifts, o.totalDuration.String())
		}
	} else {
		// Let's output it to a CSV if an output file is specified
		CSVHeaders := []string{"user", "business hours", "afterhours", "weekend hours", "stat day hours", "total hours", "shifts", "total duration oncall"}

		oFile, err := os.Create(outputFile)
		if err != nil {
			log.Fatal("Failed to create CSV output file on filesystem: ", err)
		}
		defer oFile.Close()
		writer := csv.NewWriter(oFile)
		defer writer.Flush()

		// Add all the output to a multidimensional array of strings for easy CSV printing
		csv := [][]string{CSVHeaders}
		for user, o := range fo {
			line := []string{user, strconv.Itoa(o.businessHours), strconv.Itoa(o.afterHours), strconv.Itoa(o.weekendHours),
				strconv.Itoa(o.statHours), strconv.Itoa(o.totalHours), strconv.Itoa(o.totalShifts), calendar.SheetDurationFormat(o.totalDuration)}
			csv = append(csv, line)
		}
		// Send to the csv writer
		for _, data := range csv {
			err := writer.Write(data)
			if err != nil {
				log.Fatal("Failed to write line to CSV: ", err)
			}
		}

	}

}
