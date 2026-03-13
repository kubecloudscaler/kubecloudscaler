// Package utils provides type definitions for GCP resource management.
package utils

import (
	"errors"

	compute "cloud.google.com/go/compute/apiv1"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GcpResource represents a GCP resource configuration.
type GcpResource struct {
	ProjectID string
	Region    string
	Zone      string
	Period    *periodPkg.Period `json:"period,omitempty"`
}

// ClientSet groups all GCP compute clients used by the scaler.
// Callers must call Close() when the ClientSet is no longer needed.
type ClientSet struct {
	Instances      *compute.InstancesClient
	ZoneOperations *compute.ZoneOperationsClient
	Regions        *compute.RegionsClient
}

// Close releases all underlying GCP client connections.
func (cs *ClientSet) Close() error {
	var errs []error
	if cs.Instances != nil {
		if err := cs.Instances.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if cs.ZoneOperations != nil {
		if err := cs.ZoneOperations.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if cs.Regions != nil {
		if err := cs.Regions.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Config defines the configuration for GCP resource management.
type Config struct {
	ProjectID         string                `json:"projectId,omitempty"`
	Region            string                `json:"region,omitempty"`
	Client            *ClientSet            `json:"client"`
	Names             []string              `json:"names,omitempty"`
	LabelSelector     *metaV1.LabelSelector `json:"labelSelector,omitempty"`
	Period            *periodPkg.Period     `json:"period,omitempty"`
	WaitForOperation  bool                  `json:"waitForOperation,omitempty"`
	DryRun            bool                  `json:"dryRun,omitempty"`
	DefaultPeriodType string                `json:"defaultPeriodType,omitempty"`
}
