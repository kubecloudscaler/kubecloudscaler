package utils

import (
	compute "cloud.google.com/go/compute/apiv1"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GcpResource struct {
	ProjectId string
	Region    string
	Zone      string
	Period    *periodPkg.Period `json:"period,omitempty"`
}

// ClientSet groups all GCP compute clients used by the scaler
type ClientSet struct {
	Instances      *compute.InstancesClient
	ZoneOperations *compute.ZoneOperationsClient
	Regions        *compute.RegionsClient
}

type Config struct {
	ProjectId        string                `json:"projectId,omitempty"`
	Region           string                `json:"region,omitempty"`
	Client           *ClientSet            `json:"client"`
	Names            []string              `json:"names,omitempty"`
	LabelSelector    *metaV1.LabelSelector `json:"labelSelector,omitempty"`
	Period           *periodPkg.Period     `json:"period,omitempty"`
	WaitForOperation bool                  `json:"waitForOperation,omitempty"`
	DryRun           bool                  `json:"dryRun,omitempty"`
}
