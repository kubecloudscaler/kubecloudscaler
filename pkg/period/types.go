// Package period provides type definitions for period management.
package period

import (
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
)

// Clock abstracts time.Now for testability of period evaluation.
type Clock interface {
	Now() time.Time
}

// SystemClock uses the real system time.
type SystemClock struct{}

// Now returns the current system time.
func (SystemClock) Now() time.Time { return time.Now() }

// Period represents a scaling period configuration.
type Period struct {
	Spec         *common.RecurringPeriod
	OriginalTime common.TimePeriod
	Name         string
	Type         common.PeriodType
	IsActive     bool
	Hash         string
	StartTime    time.Time
	EndTime      time.Time
	GracePeriod  time.Duration
	Once         *bool
	MinReplicas  int32
	MaxReplicas  int32
}
