package period

import (
	"time"

	"github.com/cloudscalerio/cloudscaler/api/common"
)

type Period struct {
	Period       *common.ScalerPeriod
	IsActive     bool
	Hash         string
	GetStartTime time.Time
	GetEndTime   time.Time
}
