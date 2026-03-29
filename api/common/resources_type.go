// Package common provides shared API Schema definitions for the kubecloudscaler project.
// +kubebuilder:object:generate=true
//
//nolint:revive // Package name 'common' is appropriate for shared API types
package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceKind represents a type of scalable resource.
type ResourceKind string

const (
	ResourceDeployments   ResourceKind = "deployments"
	ResourceStatefulSets  ResourceKind = "statefulsets"
	ResourceCronJobs      ResourceKind = "cronjobs"
	ResourceGithubARS     ResourceKind = "github-ars"
	ResourceHPA           ResourceKind = "hpa"
	ResourceScaledObjects ResourceKind = "scaledobjects"
	ResourceVMInstances   ResourceKind = "vm-instances"
)

// Resources defines the configuration for managed resources.
type Resources struct {
	// Types of resources
	// K8s: deployments, statefulsets, ... (default: deployments)
	// GCP: VM-instances, ... (default: vm-instances)
	Types []ResourceKind `json:"types,omitempty"`
	// Names of resources to manage
	Names []string `json:"names,omitempty"`
	// Labels selectors
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}
