package vminstances

import (
	"github.com/rs/zerolog"

	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// VMnstances handles scaling of GCP Compute Engine instances
type VMnstances struct {
	Config *gcpUtils.Config
	Period *period.Period
	Logger *zerolog.Logger
}
