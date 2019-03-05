package datasources

import (
	"time"

	"github.com/leosunmo/pagertally/pkg/config"
	"github.com/leosunmo/pagertally/pkg/timespan"
	log "github.com/sirupsen/logrus"
)

// CompanyDaysDataSource is a datasource for company days
type CompanyDaysDataSource struct {
	CompanyDays []timespan.Span
}

// NewCompanyDayDataSource returns a new company day datasource
func NewCompanyDayDataSource() CompanyDaysDataSource {
	vd := CompanyDaysDataSource{}

	vd.readCompanyDaysFromConfig()

	return vd
}

// readCompanyDaysFromConfig parses the company days from config in to time.Time and creates timespans
func (vd *CompanyDaysDataSource) readCompanyDaysFromConfig() {
	spans := []timespan.Span{}
	tz, err := time.LoadLocation(config.GlobalConfig.Timezone)
	if err != nil {
		log.Fatalf("datasources/company_days: failed to parse timezone from global config, err: %s", err.Error())
	}
	for _, day := range config.GlobalConfig.CompanyDays {
		parsedDay, terr := time.ParseInLocation(config.CompanyDayDateFormat, day, tz)
		if terr != nil {
			log.Fatalf("datasources/company_days: failed to parse company day datetime, err: %s", terr.Error())
		}
		// We currently ignore any date restrictions and just create whatever is configured.
		// Ideally we wouldn't use absolute dates anyway so this will do for now.
		span := timespan.New(parsedDay, parsedDay.AddDate(0, 0, 1))
		spans = append(spans, span)
	}
	vd.CompanyDays = spans
}

// Spans returns the timespans from the CompanyDays source
func (vd CompanyDaysDataSource) Spans() []timespan.Span {
	return vd.CompanyDays
}
