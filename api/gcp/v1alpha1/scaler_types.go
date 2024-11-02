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
	"github.com/cloudscalerio/cloudscaler/api/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ScalerSpec defines the desired state of Scaler
type ScalerSpec struct {
	// Secret containing k8s config to connect to distant cluster
	// If not set, will use the incluster client
	// AuthSecretName string `json:"authSecretName,omitempty"`

	// dry-run mode
	DryRun bool `json:"dryRun,omitempty"`

	// Time
	// // run once
	// SingleRun bool `json:"singleRun,omitempty"`
	// // Reconcile interval
	// Interval int `json:"interval,omitempty"`

	// Time period to scale
	Periods []*common.ScalerPeriod `json:"periods"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// Scaler is the Schema for the scalers API
type Scaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScalerSpec          `json:"spec,omitempty"`
	Status common.ScalerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScalerList contains a list of Scaler
type ScalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Scaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Scaler{}, &ScalerList{})
}
