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

package v1alpha1

import (
	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;update;patch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;update;patch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get

// ScalerSpec defines the desired state of Scaler
type K8sSpec struct {
	// dry-run mode
	DryRun bool `json:"dryRun,omitempty"`

	// Time period to scale
	Periods []*common.ScalerPeriod `json:"periods"`

	// Resources
	// Namespaces
	Namespaces []string `json:"namespaces,omitempty"`
	// Exclude namespaces from downscaling
	ExcludeNamespaces []string `json:"excludeNamespaces,omitempty"`
	// Force exclude system namespaces
	ForceExcludeSystemNamespaces bool `json:"forceExcludeSystemNamespaces,omitempty"`
	// Resources
	Resources []string `json:"resources,omitempty"`
	// Exclude resources from downscaling
	ExcludeResources []string `json:"excludeResources,omitempty"`
	// Labels selectors
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	// Deployment time annotation
	DeploymentTimeAnnotation string `json:"deploymentTimeAnnotation,omitempty"`
	// Disable events
	DisableEvents bool `json:"disableEvents,omitempty"`
	// AuthSecret name
	AuthSecret *string `json:"authSecret,omitempty"`
	// Restore on delete
	RestoreOnDelete bool `json:"restoreOnDelete,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +genclient

// Scaler is the Schema for the scalers API
type K8s struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sSpec             `json:"spec,omitempty"`
	Status common.ScalerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScalerList contains a list of Scaler
type K8sList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8s `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8s{}, &K8sList{})
}
