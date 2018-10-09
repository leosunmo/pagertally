package config

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/ghodss/yaml"
)

const timeShortForm = "15:04"

type ScheduleConfig struct {
	Holidays       []string            `json:"holidays,omitempty"`
	BusinessHours  businessHoursStruct `json:"business_hours"`
	RoundShiftsUp  bool                `json:"round_shifts_up"`
	CalendarURL    string              `json:"ical_url"`
	Timezone       string              `json:"ical_timezone"`
	ParsedTimezone *time.Location
}

type businessHoursStruct struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// GetScheduleConfig reads the config from file and returns a ScheduleConfig
func GetScheduleConfig(configFilePath string) *ScheduleConfig {
	// Let's set some defaults for clarity
	sc := ScheduleConfig{
		RoundShiftsUp: false,
		Timezone:      time.Now().Location().String(),
	}
	sc.mustUnmarshalScheduleConfig(mustReadScheduleConfigFile(configFilePath))
	ParsedTimezone, err := time.LoadLocation(sc.Timezone)
	if err != nil {
		fmt.Println(time.Now().Local())
		panic("Failed to parse timezone in config")
	}
	sc.ParsedTimezone = ParsedTimezone
	return &sc
}

func mustReadScheduleConfigFile(filename string) []byte {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		panic("Failed to read event configuration file")
	}
	return file
}

func (sc *ScheduleConfig) mustUnmarshalScheduleConfig(y []byte) {
	err := yaml.Unmarshal(y, sc)
	if err != nil {
		panic("Failed to unmarshal event configuration file")
	}
}

// GetBusinessHours returns the business hours start and end timestamp
// with the default 2006 date
func (sc *ScheduleConfig) GetBusinessHours() (startTime time.Time, endTime time.Time) {
	var err error
	startTime, err = time.Parse(timeShortForm, sc.BusinessHours.Start)
	if err != nil {
		panic("Failed to parse business time start time")
	}
	endTime, err = time.Parse(timeShortForm, sc.BusinessHours.End)
	if err != nil {
		panic("Failed to parse business time start time")
	}
	return startTime, endTime
}

// ShiftRoundingUp returns true if round_shifts_up is set in the config
func (sc *ScheduleConfig) ShiftRoundingUp() bool {
	return sc.RoundShiftsUp
}
