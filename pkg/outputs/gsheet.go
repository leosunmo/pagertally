package outputs

import (
	"io/ioutil"
	"log"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

const colourFrac float64 = 0.00392156863

// GSheetOutput represents a Google Sheet output destination
type GSheetOutput struct {
	spreadsheetID string
	spreadsheet   *sheets.Spreadsheet
	sheetName     string
	startCoord    string
	client        *sheets.Service
}

// NewGSheetOutput returns a new Google Sheet output struct
func NewGSheetOutput(spreadsheetID string, month string, startCoord string, saFile string) *GSheetOutput {
	return &GSheetOutput{
		spreadsheetID: spreadsheetID,
		sheetName:     month,
		startCoord:    startCoord,
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

func (g *GSheetOutput) deleteBandedRanges(sheetID int64) error {
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

func (g *GSheetOutput) findBandedRange(bandRange *sheets.BandedRange) *sheets.BandedRange {
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

func (g *GSheetOutput) updateBandedRange(bandRange *sheets.BandedRange, fields string) error {

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

func (g *GSheetOutput) addBandedRange(data [][]interface{}, sheet *sheets.Sheet) error {

	gridRange := dataToGridRange(data, sheet.Properties.SheetId, 0, 1)

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

func (g *GSheetOutput) addSheet() (*sheets.Sheet, error) {
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

func (g *GSheetOutput) findSheet() *sheets.Sheet {
	for _, sheet := range g.spreadsheet.Sheets {
		if sheet.Properties.Title == g.sheetName {
			return sheet
		}
	}
	return nil
}

// Print outputs the [][]interface{} to the Google Sheet ID provided
func (g *GSheetOutput) Print(data [][]interface{}) error {
	g.getSpreadsheetFromID()
	var vr sheets.ValueRange
	for _, v := range data {
		vr.Values = append(vr.Values, v)
	}
	var sheet *sheets.Sheet
	sheet = g.findSheet()
	if sheet == nil {
		var err error
		sheet, err = g.addSheet()
		if err != nil {
			return err
		}
	}

	_, err := g.client.Spreadsheets.Values.Update(g.spreadsheetID, g.sheetName+"!"+g.startCoord, &vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}
	err = g.addBandedRange(data, sheet)
	if err != nil {
		return err
	}

	return nil
}
