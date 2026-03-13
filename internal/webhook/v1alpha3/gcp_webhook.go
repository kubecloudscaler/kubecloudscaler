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
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

var gcplog = logf.Log.WithName("gcp-resource")

// SetupGcpWebhookWithManager registers the webhook for Gcp in the manager.
func SetupGcpWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &kubecloudscalerv1alpha3.Gcp{}).
		WithValidator(&GcpCustomValidator{}).
		Complete()
}

//nolint:lll // Kubebuilder webhook annotation cannot be split across lines
// +kubebuilder:webhook:path=/validate-kubecloudscaler-cloud-v1alpha3-gcp,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubecloudscaler.cloud,resources=gcps,verbs=create;update,versions=v1alpha3,name=vgcp-v1alpha3.kb.io,admissionReviewVersions=v1

// GcpCustomValidator validates GCP CRD resources on create and update.
//
// +kubebuilder:object:generate=false
type GcpCustomValidator struct{}

func (v *GcpCustomValidator) ValidateCreate(_ context.Context, gcp *kubecloudscalerv1alpha3.Gcp) (admission.Warnings, error) {
	gcplog.Info("Validation for Gcp upon creation", "name", gcp.GetName())
	if err := v.validateGcp(gcp); err != nil {
		return nil, fmt.Errorf("gcp validation failed: %w", err)
	}
	return nil, nil
}

func (v *GcpCustomValidator) ValidateUpdate(_ context.Context, _, gcp *kubecloudscalerv1alpha3.Gcp) (admission.Warnings, error) {
	gcplog.Info("Validation for Gcp upon update", "name", gcp.GetName())
	if err := v.validateGcp(gcp); err != nil {
		return nil, fmt.Errorf("gcp validation failed: %w", err)
	}
	return nil, nil
}

func (v *GcpCustomValidator) ValidateDelete(_ context.Context, _ *kubecloudscalerv1alpha3.Gcp) (admission.Warnings, error) {
	return nil, nil
}

func (v *GcpCustomValidator) validateGcp(gcp *kubecloudscalerv1alpha3.Gcp) error {
	if gcp.Spec.Config.ProjectID == "" {
		return fmt.Errorf("config.projectId is required")
	}

	if len(gcp.Spec.Periods) == 0 {
		return fmt.Errorf("at least one period is required")
	}

	for i, p := range gcp.Spec.Periods {
		if err := validatePeriod(p, i); err != nil {
			return err
		}
	}

	return nil
}
