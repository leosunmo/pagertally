package outputs

import "time"

// Output defines a output such as terminal, CSV or GSheet
type Output interface {
	Print(data [][]interface{}) error
}

type FinalShifts map[string]finalOutput

type finalOutput struct {
	BusinessHours int
	AfterHours    int
	WeekendHours  int
	StatHours     int
	TotalHours    int
	TotalShifts   int
	TotalDuration time.Duration
}
