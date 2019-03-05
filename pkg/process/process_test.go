package process

import (
	"strings"
	"testing"
	"time"

	"github.com/leosunmo/pagertally/pkg/config"
	"github.com/leosunmo/pagertally/pkg/datasources"
	"github.com/leosunmo/pagertally/pkg/timespan"

	log "github.com/sirupsen/logrus"
)

var aklTz, _ = time.LoadLocation("Pacific/Auckland")
var timeParseString = "2006-01-02 15:04:05 -0700 MST"

const (
	firstDec = "2018-12-01 00:00:00 +1300 NZDT"
	firstJan = "2019-01-01 00:00:00 +1300 NZDT"
)

var schedStart, _ = time.Parse(timeParseString, firstDec)
var schedEnd, _ = time.Parse(timeParseString, firstJan)

var testConfig = config.Config{
	Holidays:     []string{"Christmas Day", "Boxing Day"},
	CompanyDays:  []string{"24/12/2018", "27/12/2018", "28/12/2018", "31/12/2018"},
	CalendarURL:  "http://apps.employment.govt.nz/ical/public-holidays-all.ics",
	Timezone:     "Pacific/Auckland",
	ScheduleSpan: timespan.New(schedStart, schedEnd),
	BusinessHours: config.BusinessHoursStruct{
		Start: "08:00",
		End:   "17:30",
	},
}

var userShifts = timespan.UserShifts{
	timespan.User{
		Name:     "User1",
		Location: aklTz,
	}: []timespan.Span{
		mustParseTimeSpan("2018-12-05 17:00:00 +1300 NZDT - 2018-12-06 08:00:00 +1300 NZDT"),
		mustParseTimeSpan("2018-12-06 15:30:00 +1300 NZDT - 2018-12-06 18:00:00 +1300 NZDT"),
		mustParseTimeSpan("2018-12-07 17:00:00 +1300 NZDT - 2018-12-08 21:00:00 +1300 NZDT"),
		mustParseTimeSpan("2018-12-13 17:00:00 +1300 NZDT - 2018-12-14 17:00:00 +1300 NZDT"),
	},
	timespan.User{
		Name:     "User2",
		Location: aklTz,
	}: []timespan.Span{
		mustParseTimeSpan("2018-12-03 17:00:00 +1300 NZDT - 2018-12-05 17:00:00 +1300 NZDT"),
		mustParseTimeSpan("2018-12-18 17:00:00 +1300 NZDT - 2018-12-19 17:00:00 +1300 NZDT"),
		mustParseTimeSpan("2018-12-26 17:00:00 +1300 NZDT - 2018-12-27 17:00:00 +1300 NZDT"),
		mustParseTimeSpan("2018-12-31 17:00:00 +1300 NZDT - 2019-01-01 00:00:00 +1300 NZDT"),
	},
}

func mustParseTimeSpan(rawTimeSpan string) timespan.Span {
	stringTimestamps := strings.Split(rawTimeSpan, " - ")
	if len(stringTimestamps) > 2 {
		log.Fatal("Received more than two timestamps in string to Timespan creation")
	}
	startTime, err := time.Parse(timeParseString, stringTimestamps[0])
	if err != nil {
		log.Fatal("Failed to parse timestamp ", stringTimestamps[0])
	}
	endTime, err := time.Parse(timeParseString, stringTimestamps[1])
	if err != nil {
		log.Fatal("Failed to parse timestamp ", stringTimestamps[1])
	}
	return timespan.New(startTime, endTime)
}

// TODO: Only print details when it fails
func TestShiftAttributionDurations(t *testing.T) {
	config.GlobalConfig = testConfig
	var totalDurs time.Duration
	for user, shifts := range userShifts {
		attrShifts := attributeShift(shifts, datasources.NewCompanyDayDataSource(), datasources.NewCalendarDataSource(), datasources.NewWeekendDataSource(), datasources.NewAfterHoursDataSource())
		var shiftsDuration time.Duration
		for i, shift := range shifts {
			shiftsDuration = shiftsDuration + shift.End().Sub(shift.Start())
			t.Logf("\tShift %d:\n\t%s  -  %s\n\tDuration: %s\n\n", i, shift.Start(), shift.End(), shift.End().Sub(shift.Start()))
		}
		t.Logf("\nTotal shift duration: %s\n", shiftsDuration)
		t.Logf("\n%s's breakdown:\n", user.Name)
		var attrShiftsDuration time.Duration
		for i, shift := range attrShifts {
			attrShiftsDuration = attrShiftsDuration + shift.End().Sub(shift.Start())
			t.Logf("\tAttribShift %d Type: %d:\n\t%s  -  %s\n\tDuration: %s\n\n", i, shift.SpanType, shift.Start(), shift.End(), shift.End().Sub(shift.Start()))
		}
		t.Logf("\nTotal attribShift duration: %s\n", attrShiftsDuration)
		if shiftsDuration-attrShiftsDuration != 0 {
			t.Errorf("Shifts and attrShifts mismatch, %s vs %s\n", shiftsDuration, attrShiftsDuration)
		}
		totalDurs = totalDurs + attrShiftsDuration
	}
	expectedDuration, _ := time.ParseDuration("172h30m0s")
	if totalDurs != expectedDuration {
		t.Errorf("Expected %s, got %s", expectedDuration, totalDurs)
	}

}
