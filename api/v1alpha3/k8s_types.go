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

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list

// K8sSpec defines the desired state of K8s
type K8sSpec struct {
	// dry-run mode
	DryRun bool `json:"dryRun,omitempty"`

	// Time period to scale
	Periods []common.ScalerPeriod `json:"periods"`
	// Resources
	Resources common.Resources `json:"resources"`

	Config K8sConfig `json:"config,omitempty"`
}

// K8sConfig defines the configuration for Kubernetes resource management.
type K8sConfig struct {
	// Namespaces
	Namespaces []string `json:"namespaces,omitempty"`
	// Exclude namespaces from downscaling; will be ignored if `Namespaces` is set
	ExcludeNamespaces []string `json:"excludeNamespaces,omitempty"`
	// Force exclude system namespaces
	// +kubebuilder:default:=true
	ForceExcludeSystemNamespaces bool `json:"forceExcludeSystemNamespaces,omitempty"`
	// Deployment time annotation
	DeploymentTimeAnnotation string `json:"deploymentTimeAnnotation,omitempty"`
	// Disable events
	DisableEvents bool `json:"disableEvents,omitempty"`
	// AuthSecret name
	AuthSecret *string `json:"authSecret,omitempty"`
	// Restore resource state on CR deletion (default: true)
	// +kubebuilder:default:=true
	RestoreOnDelete bool `json:"restoreOnDelete,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +genclient

// K8s is the Schema for the k8s API
type K8s struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of K8s
	// +required
	Spec K8sSpec `json:"spec"`

	// status defines the observed state of K8s
	// +optional
	Status common.ScalerStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// K8sList contains a list of K8s
type K8sList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8s `json:"items"`
}

// GetStatus returns a pointer to the status field for use with status update utilities.
func (k *K8s) GetStatus() *common.ScalerStatus {
	return &k.Status
}

func init() {
	SchemeBuilder.Register(&K8s{}, &K8sList{})
}
