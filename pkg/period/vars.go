package period

import (
	"errors"
)

var (
	weekDays                     = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat", "all"}
	ErrBadDay                    = errors.New("invalid day notation")
	ErrFixedTimeFormat           = errors.New("bad time format for fixed period")
	ErrRecurringTimeFormat       = errors.New("bad time format for recurring period")
	ErrStartAfterEnd             = errors.New("start time is after end time")
	ErrMinReplicasGreaterThanMax = errors.New("minReplicas is greater than maxReplicas")
)
