package main

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/leosunmo/pagertally/pkg/outputs"

	"github.com/leosunmo/pagertally/pkg/config"

	"github.com/leosunmo/pagertally/pkg/datasources"

	"github.com/leosunmo/pagertally/pkg/pd"
	"github.com/leosunmo/pagertally/pkg/process"
)

func main() {
	// Add ALL the log levels!
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "trace":
		log.SetLevel(log.TraceLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	// Read config from flags, ENVVARs and config file
	config.BuildConfig()

	pdClient := pd.NewPDClient(config.PDToken())

	scheduleUserShifts, err := pd.ReadShifts(pdClient, config.Schedules(), config.StartDate(), config.EndDate())
	if err != nil {
		log.Fatalf("Failed retrieving PagerDuty schedules, %s", err.Error())
	}

	results := process.ScheduleUserShifts(scheduleUserShifts,
		datasources.NewCompanyDayDataSource(),
		datasources.NewCalendarDataSource(),
		datasources.NewWeekendDataSource(),
		datasources.NewAfterHoursDataSource())
	outputData := outputs.NewOutputData(results, config.StartDate(), config.EndDate())
	outputters := config.SelectedOutputs()

	outputErrors := outputData.PrintOutput(outputters)
	if outputErrors != nil {
		log.Fatal(outputErrors)
	}
}
