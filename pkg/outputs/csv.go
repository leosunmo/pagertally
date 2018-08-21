package outputs

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/leosunmo/pagerduty-schedule/pkg/pd"
)

// CSVOutput represents a CSV output destination
type CSVOutput struct {
	finalShifts pd.Shift
	headers     []string
	outputFile  string
}

// NewCSVOutput returns a new CSV output struct
func NewCSVOutput(outFile string) *CSVOutput {
	return &CSVOutput{
		headers:    []string{"user", "business hours", "afterhours", "weekend hours", "stat day hours", "total hours", "shifts", "total duration oncall"},
		outputFile: outFile,
	}
}

func (c *CSVOutput) Print(data [][]interface{}) error {

	oFile, err := os.Create(c.outputFile)
	if err != nil {
		return fmt.Errorf("Failed to create CSV output file on filesystem: %s", err.Error())
	}
	defer oFile.Close()
	writer := csv.NewWriter(oFile)
	defer writer.Flush()
	csvFile := [][]string{c.headers}
	for _, rs := range data {
		csvRows := make([]string, 0)
		for _, d := range rs {
			csvRows = append(csvRows, fmt.Sprint(d))
		}
		csvFile = append(csvFile, csvRows)
	}

	// Send to the csv writer
	for _, data := range csvFile {
		err := writer.Write(data)
		if err != nil {
			return fmt.Errorf("Failed to write line to CSV: %s", err.Error())
		}
	}
	return nil
}
