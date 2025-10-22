// Package utils provides type definitions for Kubernetes resource management.
//
//nolint:nolintlint,revive // package name 'utils' is acceptable for K8s utility functions
package utils

import (
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sResource represents a Kubernetes resource configuration.
type K8sResource struct {
	NsList      []string
	ListOptions metaV1.ListOptions
	Period      *periodPkg.Period `json:"period,omitempty"`
}

// Config defines the configuration for Kubernetes resource management.
type Config struct {
	Namespaces                   []string              `json:"namespaces,omitempty"`
	ExcludeNamespaces            []string              `json:"excludeNamespaces,omitempty"`
	Client                       kubernetes.Interface  `json:"client"`
	LabelSelector                *metaV1.LabelSelector `json:"labelSelector,omitempty"`
	Period                       *periodPkg.Period     `json:"period,omitempty"`
	ForceExcludeSystemNamespaces bool                  `json:"forceExcludeSystemNamespaces,omitempty"`
	Names                        []string              `json:"names,omitempty"`
}
