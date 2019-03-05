package outputs

import (
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

const colourFrac float64 = 0.00392156863

// GSheetOutputter represents a Google Sheet output destination
type GSheetOutputter struct {
	spreadsheetID string
	spreadsheet   *sheets.Spreadsheet
	sheetName     string
	startCoord    string
	client        *sheets.Service
}

type sheetData struct {
	table       sheetTable
	bandedRange *sheets.BandedRange
}

type sheetTable [][]interface{}

// NewGSheetOutputter returns a new Google Sheet outputter
func NewGSheetOutputter(spreadsheetID string, saFile string) *GSheetOutputter {
	return &GSheetOutputter{
		spreadsheetID: spreadsheetID,
		startCoord:    "A1",
		client:        getSheetClient(saFile),
	}
}
func (g *GSheetOutput) getSpreadsheetFromID() error {
	var err error
	g.spreadsheet, err = g.client.Spreadsheets.Get(g.spreadsheetID).Do()
	if err != nil {
		return err
	}
	return nil
}

// Print outputs the [][]interface{} to the Google Sheet ID provided
func (g *GSheetOutputter) Print(data OutputData) error {
	var err error
	err = g.getSpreadsheetFromID()
	if err != nil {
		return fmt.Errorf("unable to retrive spreadsheet with id %s, err: %s", g.spreadsheetID, err)
	}

	// Find the dates involved and name the sheet
	g.findMonth(data)

	// Replace with a tidy function that builds a value range from the new data format
	sheetValues, err := outputDataToSheetData(data)
	if err != nil {
		return err
	}
	var vr sheets.ValueRange
	vr.Values = sheetValues.table

	// fixup end

	sheet, sheetError := g.findOrAddSheet()
	if sheetError != nil {
		return sheetError
	}

	_, err = g.client.Spreadsheets.Values.Update(g.spreadsheetID, g.sheetName+"!"+g.startCoord, &vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}
	err = g.addBandedRange(sheetValues, sheet)
	if err != nil {
		return err
	}

	return nil
}

func (g *GSheetOutputter) findMonth(data OutputData) error {
	g.sheetName = data.DateRange.Start().Month().String() + " " + strconv.Itoa(data.DateRange.Start().Year())
	return nil
}

func (g *GSheetOutputter) getSpreadsheetFromID() error {
	var err error
	g.spreadsheet, err = g.client.Spreadsheets.Get(g.spreadsheetID).Do()
	if err != nil {
		return err
	}
	return nil
}

func outputDataToSheetData(data OutputData) (sheetData, error) {

	sheetData := sheetData{}
	sheetData.buildTable(data)

	return sheetData, nil
}

func (s *sheetData) buildTable(data OutputData) error {
	var table sheetTable
	var schedules []string
	for _, sched := range data.Schedules {
		schedules = append(schedules, sched.Name)
	}
	sort.Strings(schedules)
	// Add Schedules at the top
	schedulesString := make([]interface{}, 1)
	schedulesString[0] = strings.Join(schedules, " & ")
	table.addRow(schedulesString)
	// Add headers
	headers := []interface{}{"User", "BusinessHours", "AfterHours", "Weekend", "StatDays", "CompanyDays", "Total"}
	table.addRow(headers)

	// Crunch the user data per schedule and combine in to one table
	userDurations := map[string]TypeDurations{}
	for _, sched := range data.Schedules {
		for _, userSummary := range sched.UserShifts {
			if durs, exists := userDurations[userSummary.User.Name]; exists {
				newDurs := TypeDurations{
					OnCall:     durs.OnCall + userSummary.Durations.OnCall,
					Business:   durs.Business + userSummary.Durations.Business,
					AfterHours: durs.AfterHours + userSummary.Durations.AfterHours,
					Weekend:    durs.Weekend + userSummary.Durations.Weekend,
					Stat:       durs.Stat + userSummary.Durations.Stat,
					CompanyDay: durs.CompanyDay + userSummary.Durations.CompanyDay,
				}
				userDurations[userSummary.User.Name] = newDurs
			} else {
				userDurations[userSummary.User.Name] = userSummary.Durations
			}
		}
	}

	for user, durs := range userDurations {
		tableRow := make([]interface{}, 0)
		tableRow = append(tableRow, user)
		tableRow = append(tableRow, durs.Business)
		tableRow = append(tableRow, durs.AfterHours)
		tableRow = append(tableRow, durs.Weekend)
		tableRow = append(tableRow, durs.Stat)
		tableRow = append(tableRow, durs.CompanyDay)
		tableRow = append(tableRow, durs.OnCall)
		err := table.addRow(tableRow)
		if err != nil {
			return fmt.Errorf("unable to convert data to sheetdata, err: %s", err)
		}
	}
	// Sort the usernames in the table, minus the schedules and headers
	sort.SliceStable(table[2:], func(i, j int) bool {
		i += 2
		j += 2
		for x := range table[i] {
			if table[i][x].(string) == table[j][x].(string) {
				continue
			}
			return table[i][x].(string) < table[j][x].(string)
		}
		return false
	})
	// After we've added headers, extracted users and their on-call duration and sorted users, add to sheet
	s.table = table
	return nil
}

func getSheetClient(saFile string) *sheets.Service {
	b, err := ioutil.ReadFile(saFile)
	if err != nil {
		log.Fatalf("Unable to read service account secret file: %v", err)
	}
	config, err := google.JWTConfigFromJSON(b, sheets.SpreadsheetsScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := config.Client(context.Background())

	srv, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	return srv
}

func dataToGridRange(data [][]interface{}, sheetID int64, columnStartOffset, rowStartOffset int64) sheets.GridRange {
	var y, x int
	y = len(data)
	for _, r := range data {
		if x < len(r) {
			x = len(r)
		}
	}
	gr := sheets.GridRange{
		EndColumnIndex:   int64(x),
		EndRowIndex:      int64(y),
		StartColumnIndex: columnStartOffset,
		StartRowIndex:    rowStartOffset,
		SheetId:          sheetID,
	}

	return gr
}

func (g *GSheetOutputter) deleteBandedRanges(sheetID int64) error {
	var sheetToClean *sheets.Sheet
	for _, s := range g.spreadsheet.Sheets {
		if s.Properties.SheetId == sheetID {
			sheetToClean = s
		}
	}
	for _, br := range sheetToClean.BandedRanges {
		deleteBandingReq := sheets.DeleteBandingRequest{
			BandedRangeId: br.BandedRangeId,
		}
		req := sheets.Request{
			DeleteBanding: &deleteBandingReq,
		}
		reqs := []*sheets.Request{&req}

		batchUpdateSReq := sheets.BatchUpdateSpreadsheetRequest{
			Requests: reqs,
		}
		_, err := g.client.Spreadsheets.BatchUpdate(g.spreadsheetID, &batchUpdateSReq).Do()
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *GSheetOutputter) findBandedRange(bandRange *sheets.BandedRange) *sheets.BandedRange {
	for _, s := range g.spreadsheet.Sheets {
		for _, br := range s.BandedRanges {
			if br.Range.StartColumnIndex == bandRange.Range.StartColumnIndex &&
				br.Range.StartRowIndex == bandRange.Range.StartRowIndex {
				return br
			}
		}
	}
	return nil
}

func (g *GSheetOutputter) updateBandedRange(bandRange *sheets.BandedRange, fields string) error {

	updateBandingReq := sheets.UpdateBandingRequest{
		BandedRange: bandRange,
		Fields:      fields,
	}
	req := sheets.Request{
		UpdateBanding: &updateBandingReq,
	}
	reqs := []*sheets.Request{&req}

	batchUpdateSReq := sheets.BatchUpdateSpreadsheetRequest{
		Requests: reqs,
	}
	_, err := g.client.Spreadsheets.BatchUpdate(g.spreadsheetID, &batchUpdateSReq).Do()
	if err != nil {
		return err
	}
	return nil
}

func (g *GSheetOutputter) addBandedRange(data sheetData, sheet *sheets.Sheet) error {
	gridRange := dataToGridRange(data.table, sheet.Properties.SheetId, 0, 1)

	bandingProps := sheets.BandingProperties{
		HeaderColor:     &sheets.Color{Alpha: 0, Red: (189 * colourFrac), Green: (189 * colourFrac), Blue: (189 * colourFrac)},
		FirstBandColor:  &sheets.Color{Alpha: 0, Red: (255 * colourFrac), Green: (255 * colourFrac), Blue: (255 * colourFrac)},
		SecondBandColor: &sheets.Color{Alpha: 0, Red: (243 * colourFrac), Green: (243 * colourFrac), Blue: (243 * colourFrac)},
	}

	bandingRange := sheets.BandedRange{
		Range:         &gridRange,
		RowProperties: &bandingProps,
	}

	discoveredBandRange := g.findBandedRange(&bandingRange)
	if discoveredBandRange != nil {
		g.deleteBandedRanges(bandingRange.Range.SheetId)
	}

	addBandingReq := sheets.AddBandingRequest{
		BandedRange: &bandingRange,
	}
	req := sheets.Request{
		AddBanding: &addBandingReq,
	}
	reqs := []*sheets.Request{&req}

	batchUpdateSReq := sheets.BatchUpdateSpreadsheetRequest{
		Requests: reqs,
	}

	_, err := g.client.Spreadsheets.BatchUpdate(g.spreadsheetID, &batchUpdateSReq).Do()
	if err != nil {
		bandingRange.BandedRangeId = discoveredBandRange.BandedRangeId
		updateErr := g.updateBandedRange(&bandingRange, "*")
		if updateErr != nil {
			return updateErr
		}
	}

	return nil
}

func (g *GSheetOutputter) findOrAddSheet() (*sheets.Sheet, error) {
	var err error
	if g.sheetName == "" {
		return nil, fmt.Errorf("no sheet name provided")
	}
	sheet := g.findSheet()
	if sheet == nil {
		sheet, err = g.addSheet()
		if err != nil {
			return nil, err
		}
	}
	return sheet, nil
}

func (g *GSheetOutputter) addSheet() (*sheets.Sheet, error) {
	sheetProp := sheets.SheetProperties{
		Hidden: false,
		Title:  g.sheetName,
	}
	addSheetReq := sheets.AddSheetRequest{
		Properties: &sheetProp,
	}
	req := sheets.Request{
		AddSheet: &addSheetReq,
	}
	reqs := []*sheets.Request{&req}

	batchUpdateSReq := sheets.BatchUpdateSpreadsheetRequest{
		Requests: reqs,
	}
	_, err := g.client.Spreadsheets.BatchUpdate(g.spreadsheetID, &batchUpdateSReq).Do()
	if err != nil {
		return nil, err
	}
	var sheet *sheets.Sheet
	// Refresh our sheets so we can find the new one
	g.getSpreadsheetFromID()
	for _, s := range g.spreadsheet.Sheets {
		if s.Properties.Title == g.sheetName {
			sheet = s
		}
	}
	return sheet, nil
}

func (g *GSheetOutputter) findSheet() *sheets.Sheet {
	for _, sheet := range g.spreadsheet.Sheets {
		if sheet.Properties.Title == g.sheetName {
			return sheet
		}
	}
	return nil
}

func (t *sheetTable) addRow(row []interface{}) error {
	sanitisedRow := make([]interface{}, 0)
	for _, item := range row {
		switch item.(type) {
		default:
			return fmt.Errorf("unsupported type %+v", item)
		case nil:
			continue
		case string:
			sanitisedRow = append(sanitisedRow, item)
		case int, int64, int32:
			sanitisedRow = append(sanitisedRow, item)
		case float64:
			sanitisedRow = append(sanitisedRow, fmt.Sprintf("%.2f", item.(float64)))
		case time.Duration:
			sanitisedRow = append(sanitisedRow, sheetDurationFormat(item.(time.Duration)))
		}
	}
	*t = append(*t, sanitisedRow)
	return nil
}

// SheetDurationFormat formats the default Duration.String() string to
// Google sheets timeformat. Nanoseconds ignored.
// 48h30m25s -> 48:30:25.000
func sheetDurationFormat(d time.Duration) string {
	seconds := int64(d.Seconds()) % 60
	minutes := int64(d.Minutes()) % 60
	hours := int64(d.Hours())
	return fmt.Sprintf("%02d:%02d:%02d.000", hours, minutes, seconds)
}
