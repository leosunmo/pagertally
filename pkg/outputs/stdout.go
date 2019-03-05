package outputs

import (
	"fmt"
	"os"
	"sort"

	"github.com/olekukonko/tablewriter"
)

// StdoutOutputter will print tables to stdout
type StdoutOutputter struct {
	ShiftDetails bool
}

// NewStdoutOutput returns a StdoutOutput with shiftdetails enabled or disabled
func NewStdoutOutputter(shiftDetails bool) StdoutOutputter {
	return StdoutOutputter{
		ShiftDetails: shiftDetails,
	}
}

// Print prints one table per Schedule to Stdout
func (std StdoutOutputter) Print(data OutputData) error {
	writer := tablewriter.NewWriter(os.Stdout)
	// Schedules to Users to Durations
	tableData := map[string]map[string]TypeDurations{}
	sortedSchedules := []string{}
	// gather some useful stuff to sort on
	for _, schedule := range data.Schedules {
		sortedSchedules = append(sortedSchedules, schedule.Name)
		userDurs := map[string]TypeDurations{}
		for _, shiftSummary := range schedule.UserShifts {
			userDurs[shiftSummary.User.Name] = shiftSummary.Durations
		}
		tableData[schedule.Name] = userDurs
	}
	sort.Strings(sortedSchedules)
	for _, s := range sortedSchedules {
		fmt.Printf("Schedule: %s\n", s)
		writer.SetHeader([]string{"User", "Business Hours", "Afterhours", "Weekend", "Stat", "Company days", "Total time"})
		writer.AppendBulk(buildUsersDurationTable(tableData[s]))
		writer.Render()
		writer.ClearRows()
		fmt.Println()
	}
	return nil
}

func buildUsersDurationTable(data map[string]TypeDurations) [][]string {
	users := []string{}
	userTable := [][]string{}
	for user := range data {
		users = append(users, user)
	}
	sort.Strings(users)
	// loop over the sorted users to build the table in alphabetical order
	for _, u := range users {
		durations := data[u]
		durs := []string{u,
			durationFormat(durations.Business),
			durationFormat(durations.AfterHours),
			durationFormat(durations.Weekend),
			durationFormat(durations.Stat),
			durationFormat(durations.CompanyDay),
			durationFormat(durations.OnCall)}
		userTable = append(userTable, durs)
	}
	return userTable
}
