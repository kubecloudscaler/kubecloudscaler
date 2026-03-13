package common

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// isValidDay checks if a DayOfWeek value is valid, matching the period package's isDay logic:
// lowercase the value, take the first 3 chars, and check against known prefixes.
// This accepts "mon", "monday", "Monday", "MON", etc.
func isValidDay(day DayOfWeek) bool {
	s := strings.ToLower(string(day))
	if len(s) < dayPrefixLength {
		return false
	}
	return slices.Contains(validDayPrefixes, s[:dayPrefixLength])
}

var (
	// ErrInvalidPeriodType is returned when the period type is not "up" or "down".
	ErrInvalidPeriodType = errors.New("type must be 'up' or 'down'")
	// ErrTimeMissing is returned when neither recurring nor fixed time is set.
	ErrTimeMissing = errors.New("time must have either 'recurring' or 'fixed'")
	// ErrTimeBothSet is returned when both recurring and fixed time are set.
	ErrTimeBothSet = errors.New("time must have either 'recurring' or 'fixed', not both")
	// ErrMinGreaterThanMax is returned when minReplicas exceeds maxReplicas.
	ErrMinGreaterThanMax = errors.New("minReplicas must be <= maxReplicas")
	// ErrDaysEmpty is returned when the days list is empty.
	ErrDaysEmpty = errors.New("days must not be empty")
	// ErrInvalidDay is returned when an invalid day of week is provided.
	ErrInvalidDay = errors.New("invalid day")
	// ErrStartTimeRequired is returned when startTime is empty.
	ErrStartTimeRequired = errors.New("startTime is required")
	// ErrEndTimeRequired is returned when endTime is empty.
	ErrEndTimeRequired = errors.New("endTime is required")
)

const dayPrefixLength = 3

// validDayPrefixes is the list of valid 3-char day prefixes (lowercase).
// Matches the period package's isDay logic: lowercase + first 3 chars.
var validDayPrefixes = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// Validate checks that the ScalerPeriod configuration is valid.
func (p ScalerPeriod) Validate() error {
	if p.Type != PeriodTypeUp && p.Type != PeriodTypeDown {
		return fmt.Errorf("%w: got %q", ErrInvalidPeriodType, p.Type)
	}

	if err := p.Time.Validate(); err != nil {
		return err
	}

	if p.MinReplicas != nil && p.MaxReplicas != nil && *p.MinReplicas > *p.MaxReplicas {
		return fmt.Errorf("%w: %d > %d", ErrMinGreaterThanMax, *p.MinReplicas, *p.MaxReplicas)
	}

	return nil
}

// Validate checks that the TimePeriod configuration is valid.
func (t TimePeriod) Validate() error {
	if t.Recurring == nil && t.Fixed == nil {
		return ErrTimeMissing
	}

	if t.Recurring != nil && t.Fixed != nil {
		return ErrTimeBothSet
	}

	if t.Recurring != nil {
		return t.Recurring.Validate()
	}

	return nil
}

// Validate checks that the RecurringPeriod configuration is valid.
func (r RecurringPeriod) Validate() error {
	if len(r.Days) == 0 {
		return ErrDaysEmpty
	}

	for _, day := range r.Days {
		if !isValidDay(day) {
			return fmt.Errorf("%w: %q", ErrInvalidDay, day)
		}
	}

	if r.StartTime == "" {
		return ErrStartTimeRequired
	}

	if r.EndTime == "" {
		return ErrEndTimeRequired
	}

	return nil
}
