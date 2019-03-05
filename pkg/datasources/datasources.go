package datasources

import (
	"github.com/leosunmo/pagertally/pkg/timespan"
)

// DataSource returns a slice of Spans from it's external datasource such as a calendar of stat days
type DataSource interface {
	Spans() []timespan.Span
}
