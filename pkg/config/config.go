package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/leosunmo/pagertally/pkg/outputs"
	"github.com/leosunmo/pagertally/pkg/timespan"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const timeShortForm = "15:04"

// CompanyDayDateFormat is the date format we expect in the "company_days" config array
// TODO: support wildcard/recurring company days.
const CompanyDayDateFormat = "02/01/2006"

// GlobalConfig is the currently active configuration
var GlobalConfig Config

// SecretString is a string that prevents accidental printing
// string(mySecretString) to get value
type SecretString string

// Config is the application config
type Config struct {
	Holidays       []string            `json:"holidays,omitempty"`
	BusinessHours  BusinessHoursStruct `json:"business_hours"`
	CalendarURL    string              `json:"ical_url"`
	Timezone       string              `json:"timezone"`
	CompanyDays    []string            `json:"company_days,omitempty"`
	CsvDir         string
	ScheduleSpan   timespan.Span
	ParsedTimezone *time.Location
	Debug          bool
}

// BusinessHoursStruct is a struct of string representations of business hours start and end
type BusinessHoursStruct struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// BuildConfig parses command-line flags, config file and environment variables and builds up the application configuration
func BuildConfig() {
	var pdToken SecretString
	// First look up flags
	flag.StringSliceP("schedules", "s", nil, "Comma separated list of PagerDuty schedule IDs")
	flag.StringP("config", "c", "", "(Optional) Provide config file path. Looks for \"config.yaml\" by default")
	flag.String("csvdir", "", "(Optional) Print as CSVs to this directory")
	flag.String("gsheetid", "", "(Optional) Print to Google Sheet ID provided")
	flag.String("google-safile", "", "(Optional) Google Service Account token JSON file")
	flag.VarP(&pdToken, "pagerduty-token", "t", "PagerDuty API token")
	flag.StringP("month", "m", "", "(Optional) Provide the month and year you want to process. Format: March 2018. Default: previous month")
	printHelp := flag.BoolP("help", "h", false, "Print usage")

	// Parse flags
	flag.Parse()
	if *printHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Register some aliases
	viper.RegisterAlias("sheet-id", "gsheetid")
	viper.RegisterAlias("start-month", "month")
	viper.RegisterAlias("pagerduty-schedules", "schedules")

	// Bind the resulting flags to Viper values
	viper.BindPFlags(flag.CommandLine)

	if pdToken != "" {
		// Force the pagerduty token to the correct SecretString type
		viper.Set("pagerduty-token", pdToken)
	}

	// We prefix all ENVVARs with "PDS_" because that's good practice
	viper.SetEnvPrefix("pds")
	// Config will be a YAML file
	viper.SetConfigType("yaml")
	// Expect a file called "config.yaml" by default
	viper.SetConfigName("config")
	// Look for that file in the current directory
	viper.AddConfigPath(".")
	// This will look for any ENVVAR with the "PDS_" prefix automatically and bind them to Viper values
	viper.AutomaticEnv()
	// Make sure we use a strings.Replacer so we can convert "-" to "_" in ENVVARs
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	if insecurePdToken, exists := os.LookupEnv("PDS_PAGERDUTY_TOKEN"); exists {
		pdToken = SecretString(insecurePdToken)
		// Force the pagerduty token to the correct SecretString type
		viper.Set("pagerduty-token", pdToken)
	}

	// If we've explicitly set the config flag, use that file
	if viper.IsSet("config") {
		viper.SetConfigFile(viper.GetString("config"))
	}
	// Read config file, provided through flag or directory discovery
	err := viper.ReadInConfig()
	if err != nil {
		e, ok := err.(viper.ConfigParseError)
		if ok {
			log.Fatalf("error parsing config file: %v", e)
		}
		// Should this be a fatal? Technically you could provide everything through other means
		log.Warn("no config file found")
	}
	log.Debug("Using config file: ", viper.ConfigFileUsed())

	// Set defaults
	viper.SetDefault("timezone", time.Local.String())
	viper.SetDefault("business_hours.start", "09:00")
	viper.SetDefault("business_hours.end", "17:30")

	// Create a time.Location using the timezone that we can use for parsing
	loc, err := time.LoadLocation(viper.GetString("timezone"))
	if err != nil {
		log.Fatalf("Failed to parse timezone. use IANA TZ format, err: %s", err.Error())
	}

	// If start-month is not set, default to previous month
	if !viper.IsSet("start-month") || viper.GetString("start-month") == "" {
		viper.Set("start-month", fmt.Sprintf("%s %d", time.Now().AddDate(0, -1, 0).Month(), time.Now().AddDate(0, -1, 0).Year()))
	}

	// Create a time.Time from the start-month
	startDate, err := time.ParseInLocation("January 2006", viper.GetString("start-month"), loc)
	endDate := startDate.AddDate(0, +1, 0)

	// Let's add start and end dates to viper as well for convenience
	viper.Set("start_date", startDate)
	viper.Set("end_date", endDate)

	// fail on mandatory config
	if !viper.IsSet("pagerduty-token") || string(viper.Get("pagerduty-token").(SecretString)) == "" {
		log.Fatal("PagerDuty access token not provided. Use 'PDS_PAGERDUTY_TOKEN' or flag '--pagerduty-token' / '-t'")
	}
	if !viper.IsSet("schedules") || len(viper.GetStringSlice("schedules")) == 0 {
		log.Fatal("PagerDuty schedules not specified. Use comma separated list in envvar 'PDS_PAGERDUTY_SCHEDULES' or flag '--schedules'")
	}

	// Kind of a hack because of https://github.com/spf13/viper/issues/380
	viper.Set("schedules", commaSeparatedStringToSlice(viper.GetStringSlice("schedules")))

	if viper.IsSet("gsheetid") {
		if !viper.IsSet("google-safile") {
			log.Fatal("Google sheets output requires a Google service account file (\"--google-safile\")")
		}
	}

	viper.Set("parsed_timezone", loc)

	GlobalConfig = Config{
		Holidays: viper.GetStringSlice("holidays"),
		BusinessHours: BusinessHoursStruct{
			Start: viper.GetString("business_hours.start"),
			End:   viper.GetString("business_hours.end"),
		},
		CalendarURL:    viper.GetString("ical_url"),
		Timezone:       viper.GetString("timezone"),
		CompanyDays:    viper.GetStringSlice("company_days"),
		ParsedTimezone: viper.Get("parsed_timezone").(*time.Location),
		ScheduleSpan:   timespan.New(viper.GetTime("start_date"), viper.GetTime("end_date")),
		Debug:          viper.GetBool("debug"),
	}

	log.Debug(fmt.Sprintf("Viper Configuration: %+v", viper.AllSettings()))
}

// BusinessHoursForDate returns the business hours start and end timestamp
// by taking the provided day's date combined with the configured tz
func BusinessHoursForDate(day time.Time) (startTime time.Time, endTime time.Time) {
	var err error
	var start, end time.Time
	refDate := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, Timezone())
	startTime, err = time.ParseInLocation(timeShortForm, GlobalConfig.BusinessHours.Start, refDate.Location())
	if err != nil {
		log.Fatalf("failed to parse business hour time, string: %s, layout: %s", GlobalConfig.BusinessHours.Start, timeShortForm)
	}
	endTime, err = time.ParseInLocation(timeShortForm, GlobalConfig.BusinessHours.End, refDate.Location())
	if err != nil {
		log.Fatalf("failed to parse business hour time, string: %s, layout: %s", GlobalConfig.BusinessHours.End, timeShortForm)
	}
	start = refDate.Add((time.Hour * time.Duration(startTime.Hour())) + (time.Minute * time.Duration(startTime.Minute())))
	end = refDate.Add((time.Hour * time.Duration(endTime.Hour())) + (time.Minute * time.Duration(endTime.Minute())))
	return start, end
}

// Timezone returns the configured local timezone
func Timezone() *time.Location {
	if GlobalConfig.ParsedTimezone != nil {
		return GlobalConfig.ParsedTimezone
	}
	loc, err := time.LoadLocation(GlobalConfig.Timezone)
	if err != nil {
		log.Fatal("config/config.go failed to parse timezone")
	}
	GlobalConfig.ParsedTimezone = loc
	return loc
}

// SelectedOutputs returns a list of all configured outputs
func SelectedOutputs() []outputs.Outputter {
	var o []outputs.Outputter
	// We can't simply use IsSet() here as there's a bug preventing us when we've bound to a pFlag, https://github.com/spf13/viper/issues/276
	if viper.IsSet("csvdir") && viper.GetString("csvdir") != "" {
		o = append(o, outputs.NewCSVOutputter(viper.GetString("csvdir")))
	}
	if viper.IsSet("gsheetid") && viper.GetString("gsheetid") != "" {
		o = append(o, outputs.NewGSheetOutputter(viper.GetString("gsheetid"), viper.GetString("google-safile")))
	}
	if len(o) < 1 {
		o = append(o, outputs.NewStdoutOutputter(false))
	}
	return o
}

// Schedules returns all configured PagerDuty schedules
func Schedules() []string {
	return viper.GetStringSlice("schedules")
}

// StartDate returns the configured startdate
func StartDate() time.Time {
	return viper.GetTime("start_date")
}

// EndDate returns the configured startdate
func EndDate() time.Time {
	return viper.GetTime("end_date")
}

// PDToken returns the configured PagerDuty API token
func PDToken() string {
	return string(viper.Get("pagerduty-token").(SecretString))
}

// GToken returns the configured Google Service Account token file location
func GToken() string {
	return viper.GetString("google-safile")
}

func commaSeparatedStringToSlice(s []string) []string {
	if len(s) > 1 {
		return s
	}
	return strings.Split(s[0], ",")
}

// String method for SecretString stringer that prevents printing credentials
func (s SecretString) String() string {
	return "[REDACTED]"
}

// Set method for SecretString
func (s *SecretString) Set(newValue string) error {
	*s = SecretString(newValue)
	return nil
}

// Type method for SecretString
func (s SecretString) Type() string {
	return "SecretString"
}

// ShiftRoundingUp returns true if round_shifts_up is set in the config
func (sc *ScheduleConfig) ShiftRoundingUp() bool {
	return sc.RoundShiftsUp
}
