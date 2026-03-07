// Package period provides constants for period management.
package period

const (
	// AllDays represents "all" for recurring periods spanning every day.
	AllDays = "all"
	// NoactionPeriodName is the name/type for the "no active period" fallback.
	NoactionPeriodName = "noaction"
	// PeriodFixedName is the name for fixed period type.
	PeriodFixedName = "fixed"
	// PeriodRecurringName is the name for recurring period type.
	PeriodRecurringName = "recurring"
	defaultTimezone     = "UTC"
	defaultGracePeriod  = "0s"
	dayStringLength     = 3
)
