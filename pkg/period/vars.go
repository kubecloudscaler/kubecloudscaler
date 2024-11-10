package period

import (
	"errors"
)

var (
	weekDays               = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat", "all"}
	ErrBadDay              = errors.New("invalid day notation")
	ErrFixedTimeFormat     = errors.New("bad time format for fixed period")
	ErrRecurringTimeFormat = errors.New("bad time format for recurring period")
)
