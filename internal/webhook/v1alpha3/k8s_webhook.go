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

var k8slog = logf.Log.WithName("k8s-resource")

// SetupK8sWebhookWithManager registers the webhook for K8s in the manager.
func SetupK8sWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &kubecloudscalerv1alpha3.K8s{}).
		WithValidator(&K8sCustomValidator{}).
		Complete()
}

//nolint:lll // Kubebuilder webhook annotation cannot be split across lines
// +kubebuilder:webhook:path=/validate-kubecloudscaler-cloud-v1alpha3-k8s,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubecloudscaler.cloud,resources=k8s,verbs=create;update,versions=v1alpha3,name=vk8s-v1alpha3.kb.io,admissionReviewVersions=v1

// K8sCustomValidator validates K8s CRD resources on create and update.
//
// +kubebuilder:object:generate=false
type K8sCustomValidator struct{}

func (v *K8sCustomValidator) ValidateCreate(_ context.Context, k8s *kubecloudscalerv1alpha3.K8s) (admission.Warnings, error) {
	k8slog.Info("Validation for K8s upon creation", "name", k8s.GetName())
	if err := v.validateK8s(k8s); err != nil {
		return nil, fmt.Errorf("k8s validation failed: %w", err)
	}
	return nil, nil
}

func (v *K8sCustomValidator) ValidateUpdate(_ context.Context, _, k8s *kubecloudscalerv1alpha3.K8s) (admission.Warnings, error) {
	k8slog.Info("Validation for K8s upon update", "name", k8s.GetName())
	if err := v.validateK8s(k8s); err != nil {
		return nil, fmt.Errorf("k8s validation failed: %w", err)
	}
	return nil, nil
}

func (v *K8sCustomValidator) ValidateDelete(_ context.Context, _ *kubecloudscalerv1alpha3.K8s) (admission.Warnings, error) {
	return nil, nil
}

func (v *K8sCustomValidator) validateK8s(k8s *kubecloudscalerv1alpha3.K8s) error {
	if len(k8s.Spec.Periods) == 0 {
		return fmt.Errorf("at least one period is required")
	}

	for i, p := range k8s.Spec.Periods {
		if err := validatePeriod(p, i); err != nil {
			return err
		}
	}

	return nil
}
