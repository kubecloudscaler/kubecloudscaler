// Package instancegroupmanagers provides MIG scaling functionality for GCP resources.
package instancegroupmanagers

import (
	"github.com/rs/zerolog"

	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// InstanceGroupManagers handles scaling of GCP Managed Instance Group (MIG) instances
// via the MIG-aware stopInstances/startInstances API. This preserves boot disks and
// prevents the autohealer from recreating stopped instances (unlike instances.stop).
type InstanceGroupManagers struct {
	Config *gcpUtils.Config
	Period *period.Period
	Logger *zerolog.Logger
}
