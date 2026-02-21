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

// GcpSpec defines the desired state of Gcp
type GcpSpec struct {
	// Secret containing k8s config to connect to distant cluster
	// If not set, will use the incluster client
	// AuthSecretName string `json:"authSecretName,omitempty"`

	// dry-run mode
	DryRun bool `json:"dryRun,omitempty"`

	// Time period to scale
	Periods []common.ScalerPeriod `json:"periods"`
	// Resources
	Resources common.Resources `json:"resources"`

	Config GcpConfig `json:"config,omitempty"`
}

// GcpConfig defines the configuration for GCP resource management.
type GcpConfig struct {
	// ProjectID
	ProjectID string `json:"projectId"`
	// Region
	Region string `json:"region,omitempty"`
	// AuthSecret name
	AuthSecret *string `json:"authSecret,omitempty"`
	// RestoreOnDelete applies defaultPeriodType to all managed resources when the CR is deleted.
	// Note: this does NOT restore the pre-CR state of resources. It applies the defaultPeriodType
	// value (default: "down"), meaning VMs will be stopped on deletion unless defaultPeriodType
	// is set to "up". To restore VMs to their original state, set defaultPeriodType accordingly.
	// +kubebuilder:default:=true
	RestoreOnDelete bool `json:"restoreOnDelete,omitempty"`
	// Wait for operation to complete
	WaitForOperation bool `json:"waitForOperation,omitempty"`
	// Default status for resources
	// +kubebuilder:validation:Enum=down;up
	// +kubebuilder:default:=down
	DefaultPeriodType string `json:"defaultPeriodType,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +genclient

// Gcp is the Schema for the gcps API
type Gcp struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Gcp
	// +required
	Spec GcpSpec `json:"spec"`

	// status defines the observed state of Gcp
	// +optional
	Status common.ScalerStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// GcpList contains a list of Gcp
type GcpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gcp `json:"items"`
}

// GetStatus returns a pointer to the status field for use with status update utilities.
func (g *Gcp) GetStatus() *common.ScalerStatus {
	return &g.Status
}

func init() {
	SchemeBuilder.Register(&Gcp{}, &GcpList{})
}
