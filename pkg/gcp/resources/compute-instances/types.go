package computeinstances

import (
	"github.com/rs/zerolog"

	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// ComputeInstances handles scaling of GCP Compute Engine instances
type ComputeInstances struct {
	Config *gcpUtils.Config
	Period *period.Period
	Logger *zerolog.Logger
}
