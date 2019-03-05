package outputs

import (
	"fmt"
	"sort"
	"time"

	"github.com/leosunmo/pagertally/pkg/timespan"
)

// Outputter defines a output such as terminal, CSV or GSheet
type Outputter interface {
	Print(OutputData) error
}

// OutputData is a struct with the final data ready to output
type OutputData struct {
	RawResults map[string][]timespan.UserShiftResults // RawResults is a map of schedule names to users and their shifts
	DateRange  timespan.Span
	Schedules  []Schedule
}

type Schedule struct {
	Name       string
	UserShifts []ShiftsSummary
}

type ShiftsSummary struct {
	User             UserDetails
	AttributedShifts []AttributedShiftSpans
	Durations        TypeDurations
	CompanyDays      int
}

type UserDetails struct {
	Name     string
	Timezone *time.Location
}

type TypeDurations struct {
	OnCall     time.Duration
	Business   time.Duration
	AfterHours time.Duration
	Weekend    time.Duration
	Stat       time.Duration
	CompanyDay time.Duration
}

// AttributedShiftSpans is a map of the shift span to it's associated attributed spans
type AttributedShiftSpans map[timespan.Span]timespan.AttributedSpans

// NewOutputData returns a new OutputData with the final data ready for easy use
func NewOutputData(results map[string][]timespan.UserShiftResults, startDate, endDate time.Time) OutputData {
	data := OutputData{
		RawResults: results,
		DateRange:  timespan.New(startDate, endDate),
		Schedules:  []Schedule{},
	}

	for schedName, userResults := range results {
		userShiftSummary := []ShiftsSummary{}
		for _, userResult := range userResults {
			userShifts := ShiftsSummary{
				User: UserDetails{
					Name:     userResult.User.Name,
					Timezone: userResult.User.Location,
				},
				AttributedShifts: buildAttributedShiftSpans(userResult),
				Durations:        buildDurations(userResult),
				CompanyDays:      userResult.Breakdown.CompanyDayCount(),
			}
			userShiftSummary = append(userShiftSummary, userShifts)
		}
		schedule := Schedule{
			Name:       schedName,
			UserShifts: userShiftSummary,
		}
		data.Schedules = append(data.Schedules, schedule)
	}
	return data
}

// PrintOutput runs the Print methods on all provided outputters
func (data OutputData) PrintOutput(outputs []Outputter) []error {
	var errors []error
	for _, output := range outputs {
		err := output.Print(data)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) != 0 {
		return errors
	}
	return nil
}

func buildAttributedShiftSpans(shiftResults timespan.UserShiftResults) []AttributedShiftSpans {
	output := []AttributedShiftSpans{}
	// Iterate over all the shift spans and find it's attributed spans
	for _, shiftSpan := range shiftResults.Shifts {
		// associatedSpans for all attributed spans
		associatedSpans := timespan.AttributedSpans{}
		//Find the matching attributed spans
		for _, attribSpan := range shiftResults.Breakdown {
			if attribSpan.Overlaps(shiftSpan) {
				associatedSpans = append(associatedSpans, attribSpan)
			}
		}
		// Sort the associated spans just in case
		sort.Sort(associatedSpans)

		// attribShift for a single shift to multiple attributed spans
		attribShift := AttributedShiftSpans{}
		attribShift[shiftSpan] = associatedSpans
		output = append(output, attribShift)
	}
	return output
}

func buildDurations(results timespan.UserShiftResults) TypeDurations {
	return TypeDurations{
		OnCall:     results.Breakdown.TotalDur(),
		Business:   results.Breakdown.BusinessHoursDur(),
		AfterHours: results.Breakdown.AfterHoursDur(),
		Weekend:    results.Breakdown.WeekendDur(),
		Stat:       results.Breakdown.StatDur(),
		CompanyDay: results.Breakdown.CompanyDayDur(),
	}

}

func durationFormat(d time.Duration) string {
	if d < 1 {
		return "-"
	}
	seconds := int64(d.Seconds()) % 60
	minutes := int64(d.Minutes()) % 60
	hours := int64(d.Hours())
	if seconds == 0 {
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
}
