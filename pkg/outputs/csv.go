package outputs

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CSVOutputter outputs one CSV file per schedule to the filesystem
type CSVOutputter struct {
	outputLocation string
}

type csvFile [][]string

// NewCSVOutputter returns a new CSV outputter
func NewCSVOutputter(outLocation string) *CSVOutputter {
	return &CSVOutputter{
		outputLocation: outLocation,
	}
}

// Print outputs the data to a CSV writer
func (c *CSVOutputter) Print(data OutputData) error {

	for _, sched := range data.Schedules {
		var csvFile csvFile
		normalisedName := strings.Replace(strings.ToLower(sched.Name), " ", "_", -1) + ".csv"
		oFile, err := os.Create(filepath.Clean(c.outputLocation + normalisedName))
		if err != nil {
			return fmt.Errorf("Failed to create CSV output file on filesystem: %s", err.Error())
		}
		defer oFile.Close()
		// Add headers
		headers := []interface{}{"User", "BusinessHours", "AfterHours", "Weekend", "StatDays", "CompanyDays", "Total"}
		csvFile.addRow(headers)
		for _, shift := range sched.UserShifts {
			csvRow := make([]interface{}, 0)
			csvRow = append(csvRow, shift.User.Name)
			csvRow = append(csvRow, shift.Durations.Business)
			csvRow = append(csvRow, shift.Durations.AfterHours)
			csvRow = append(csvRow, shift.Durations.Weekend)
			csvRow = append(csvRow, shift.Durations.Stat)
			csvRow = append(csvRow, shift.Durations.CompanyDay)
			csvRow = append(csvRow, shift.Durations.OnCall)
			csvFile.addRow(csvRow)
		}

		// Semi hack to sort by username but avoiding the headers
		sort.SliceStable(csvFile[1:], func(i, j int) bool {
			i++
			j++
			for x := range csvFile[i] {
				if csvFile[i][x] == csvFile[j][x] {
					continue
				}
				return csvFile[i][x] < csvFile[j][x]
			}
			return false
		})

		// Send to the csv writer
		writer := csv.NewWriter(oFile)
		defer writer.Flush()
		for _, finalRow := range csvFile {
			err := writer.Write(finalRow)
			if err != nil {
				return fmt.Errorf("Failed to write line to CSV: %s", err.Error())
			}
		}
	}
	return nil
}

func (cf *csvFile) addRow(row []interface{}) error {
	sanitisedRow := []string{}
	for _, item := range row {
		switch item.(type) {
		default:
			return fmt.Errorf("unsupported csv type %+v", item)
		case nil:
			continue
		case string:
			sanitisedRow = append(sanitisedRow, item.(string))
		case int, int64, int32:
			sanitisedRow = append(sanitisedRow, strconv.Itoa(item.(int)))
		case float64:
			sanitisedRow = append(sanitisedRow, fmt.Sprintf("%.2f", item.(float64)))
		case time.Duration:
			sanitisedRow = append(sanitisedRow, durationFormat(item.(time.Duration)))
		}
	}
	*cf = append(*cf, sanitisedRow)
	return nil
}
