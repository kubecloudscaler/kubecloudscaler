package period

import (
	"time"

	"github.com/cloudscalerio/cloudscaler/api/common"
)

type Period struct {
	Period       *common.RecurringPeriod
	Type         string
	IsActive     bool
	Hash         string
	GetStartTime time.Time
	GetEndTime   time.Time
	Once         *bool
	MinReplicas  *int32
	MaxReplicas  *int32
}
