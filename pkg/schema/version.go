package schema

// SchemaVersion is the version of the JSON documents produced by price-checker.
// See the package documentation for the compatibility policy.
const SchemaVersion = "1.0"

// MonthlyHours is the number of hours price-checker uses to convert an hourly
// rate to a monthly cost. It matches the convention used by AWS and Infracost.
const MonthlyHours = 730

// Period identifies which cost column a caller emphasises. It does not change
// the values in a document (both monthly and hourly are always populated); it
// is carried in the document so renderers know the user's preference.
type Period string

const (
	PeriodMonth Period = "month"
	PeriodHour  Period = "hour"
)

// Valid reports whether p is a recognised period.
func (p Period) Valid() bool { return p == PeriodMonth || p == PeriodHour }
