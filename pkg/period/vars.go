package period

import (
	"errors"
)

var (
	weekDays   = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat", "all"}
	ErrBadDay  = errors.New("invalid day notation")
	ErrBadTime = errors.New("invalid time notation")
)
