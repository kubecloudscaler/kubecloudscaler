// Package period provides variables and constants for period management.
package period

import (
	"errors"
)

var (
	weekDays = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat", "all"}
	// ErrBadDay is returned when an invalid day notation is provided.
	ErrBadDay = errors.New("invalid day notation")
	// ErrFixedTimeFormat is returned when the time format for fixed period is invalid.
	ErrFixedTimeFormat = errors.New("bad time format for fixed period")
	// ErrRecurringTimeFormat is returned when the time format for recurring period is invalid.
	ErrRecurringTimeFormat = errors.New("bad time format for recurring period")
	// ErrStartAfterEnd is returned when the start time is after the end time.
	ErrStartAfterEnd = errors.New("start time is after end time")
	// ErrMinReplicasGreaterThanMax is returned when min replicas is greater than max replicas.
	ErrMinReplicasGreaterThanMax = errors.New("minReplicas is greater than maxReplicas")
	// ErrUnknownPeriodType is returned when an unknown period type is provided.
	ErrUnknownPeriodType = errors.New("unknown period type")
)
