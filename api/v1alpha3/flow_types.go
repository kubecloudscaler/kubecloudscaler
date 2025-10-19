/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FlowSpec defines the desired state of Flow
type FlowSpec struct {
	// Time period to scale
	Periods []common.ScalerPeriod `json:"periods"`
	// Resources
	Resources Resources `json:"resources"`
	Flows     []Flows   `json:"flows,omitempty"`
}

type Resources struct {
	K8s []K8sResource `json:"k8s,omitempty"`
	Gcp []GcpResource `json:"gcp,omitempty"`
}

type K8sResource struct {
	Name      string           `json:"name"`
	Resources common.Resources `json:"resources"`
	Config    K8sConfig        `json:"config,omitempty"`
}

type GcpResource struct {
	Name      string           `json:"name"`
	Resources common.Resources `json:"resources"`
	Config    GcpConfig        `json:"config,omitempty"`
}

type Flows struct {
	PeriodName string         `json:"periodName"`
	Resources  []FlowResource `json:"resources"`
}

type FlowResource struct {
	Name string `json:"name"`

	// Delay is the duration to delay the start of the period
	// It is a duration in minutes
	// It is optional and if not provided, the period will start at the start time of the period
	// +kubebuilder:validation:Pattern=`^\d*m$`
	Delay *string `json:"delay,omitempty"`
}

// FlowStatus defines the observed state of Flow.
type FlowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Flow resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// Flow is the Schema for the flows API
type Flow struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Flow
	// +required
	Spec FlowSpec `json:"spec"`

	// status defines the observed state of Flow
	// +optional
	Status FlowStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// FlowList contains a list of Flow
type FlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Flow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Flow{}, &FlowList{})
}
