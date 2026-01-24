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

// Package v1alpha3 provides webhook functionality for GCP resources in the kubecloudscaler API.
package v1alpha3

import (
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

// log is for logging in this package.
//
//nolint:unused // Variable is used for logging in webhook operations
var gcplog = logf.Log.WithName("gcp-resource")

// SetupGcpWebhookWithManager registers the webhook for Gcp in the manager.
func SetupGcpWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &kubecloudscalerv1alpha3.Gcp{}).
		Complete()
}
