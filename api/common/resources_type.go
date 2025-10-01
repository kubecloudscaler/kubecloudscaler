// +kubebuilder:object:generate=true
package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Resources struct {
	// Types of resources
	// K8s: deployments, statefulsets, ... (default: deployments)
	// GCP: VM-instances, ... (default: vm-instances)
	Types []string `json:"types,omitempty"`
	// Names of resources to manage
	Names []string `json:"names,omitempty"`
	// Labels selectors
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}
