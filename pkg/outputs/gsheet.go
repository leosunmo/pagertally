package outputs

import (
	"io/ioutil"
	"log"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

// GSheetOutput represents a Google Sheet output destination
type GSheetOutput struct {
	spreadsheetID string
	startCoord    string
	headers       []interface{}
	client        *sheets.Service
}

// NewGSheetOutput returns a new Google Sheet output struct
func NewGSheetOutput(spreadsheetID string, startCoord string, saFile string) *GSheetOutput {
	return &GSheetOutput{
		spreadsheetID: spreadsheetID,
		startCoord:    startCoord,
		headers:       []interface{}{"user", "business hours", "afterhours", "weekend hours", "stat day hours", "total hours", "shifts", "total duration oncall"},
		client:        getSheetClient(saFile),
	}
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

// Print outputs the [][]interface{} to the Google Sheet ID provided
func (g *GSheetOutput) Print(data [][]interface{}) error {

	var vr sheets.ValueRange
	vr.Values = append(vr.Values, g.headers)
	for _, v := range data {
		vr.Values = append(vr.Values, v)
	}
	_, err := g.client.Spreadsheets.Values.Update(g.spreadsheetID, g.startCoord, &vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return err
	}
	return nil
}
