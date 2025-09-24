package utils

import (
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	compute "google.golang.org/api/compute/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GcpResource struct {
	ProjectId string
	Region    string
	Zone      string
	Period    *periodPkg.Period `json:"period,omitempty"`
}

type Config struct {
	ProjectId        string                `json:"projectId,omitempty"`
	Region           string                `json:"region,omitempty"`
	Client           *compute.Service      `json:"client"`
	Resources        []string              `json:"resources,omitempty"`
	ExcludeResources []string              `json:"excludeResources,omitempty"`
	LabelSelector    *metaV1.LabelSelector `json:"labelSelector,omitempty"`
	Period           *periodPkg.Period     `json:"period,omitempty"`
}
