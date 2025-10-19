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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalercloudv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

// nolint:unused
// log is for logging in this package.
var flowlog = logf.Log.WithName("flow-resource")

// SetupFlowWebhookWithManager registers the webhook for Flow in the manager.
func SetupFlowWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&kubecloudscalercloudv1alpha3.Flow{}).
		WithValidator(&FlowCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-kubecloudscaler-cloud-v1alpha3-flow,mutating=false,failurePolicy=fail,sideEffects=None,groups=kubecloudscaler.cloud,resources=flows,verbs=create;update,versions=v1alpha3,name=vflow-v1alpha3.kb.io,admissionReviewVersions=v1

// FlowCustomValidator struct is responsible for validating the Flow resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type FlowCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &FlowCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Flow.
func (v *FlowCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	flow, ok := obj.(*kubecloudscalercloudv1alpha3.Flow)
	if !ok {
		return nil, fmt.Errorf("expected a Flow object but got %T", obj)
	}
	flowlog.Info("Validation for Flow upon creation", "name", flow.GetName())

	// Perform comprehensive validation
	if err := v.validateFlow(flow); err != nil {
		flowlog.Error(err, "Flow validation failed during creation", "name", flow.GetName())
		return nil, fmt.Errorf("flow validation failed: %w", err)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Flow.
func (v *FlowCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	flow, ok := newObj.(*kubecloudscalercloudv1alpha3.Flow)
	if !ok {
		return nil, fmt.Errorf("expected a Flow object for the newObj but got %T", newObj)
	}
	flowlog.Info("Validation for Flow upon update", "name", flow.GetName())

	// Perform comprehensive validation
	if err := v.validateFlow(flow); err != nil {
		flowlog.Error(err, "Flow validation failed during update", "name", flow.GetName())
		return nil, fmt.Errorf("flow validation failed: %w", err)
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Flow.
func (v *FlowCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	flow, ok := obj.(*kubecloudscalercloudv1alpha3.Flow)
	if !ok {
		return nil, fmt.Errorf("expected a Flow object but got %T", obj)
	}
	flowlog.Info("Validation for Flow upon deletion", "name", flow.GetName())

	// No validation needed for deletion
	return nil, nil
}

// validateFlow performs comprehensive validation of the flow definition
func (v *FlowCustomValidator) validateFlow(flow *kubecloudscalercloudv1alpha3.Flow) error {
	// Validate period name uniqueness
	if err := v.validatePeriodNameUniqueness(flow); err != nil {
		return fmt.Errorf("period validation failed: %w", err)
	}

	// Validate resource name uniqueness
	if err := v.validateResourceNameUniqueness(flow); err != nil {
		return fmt.Errorf("resource validation failed: %w", err)
	}

	// Validate flow timings
	if err := v.validateFlowTimings(flow); err != nil {
		return fmt.Errorf("timing validation failed: %w", err)
	}

	return nil
}

// validatePeriodNameUniqueness ensures each period has a unique name
func (v *FlowCustomValidator) validatePeriodNameUniqueness(flow *kubecloudscalercloudv1alpha3.Flow) error {
	periodNames := make(map[string]bool)

	for i, period := range flow.Spec.Periods {
		periodName := ptr.Deref(period.Name, "")
		if periodName == "" {
			return fmt.Errorf("period at index %d has no name", i)
		}

		if periodNames[periodName] {
			return fmt.Errorf("duplicate period name '%s' found", periodName)
		}

		periodNames[periodName] = true
	}

	return nil
}

// validateResourceNameUniqueness ensures each resource has a unique name within its type
func (v *FlowCustomValidator) validateResourceNameUniqueness(flow *kubecloudscalercloudv1alpha3.Flow) error {
	// Validate K8s resource names
	k8sNames := make(map[string]bool)
	for i, resource := range flow.Spec.Resources.K8s {
		if resource.Name == "" {
			return fmt.Errorf("K8s resource at index %d has no name", i)
		}

		if k8sNames[resource.Name] {
			return fmt.Errorf("duplicate K8s resource name '%s' found", resource.Name)
		}

		k8sNames[resource.Name] = true
	}

	// Validate GCP resource names
	gcpNames := make(map[string]bool)
	for i, resource := range flow.Spec.Resources.Gcp {
		if resource.Name == "" {
			return fmt.Errorf("GCP resource at index %d has no name", i)
		}

		if gcpNames[resource.Name] {
			return fmt.Errorf("duplicate GCP resource name '%s' found", resource.Name)
		}

		gcpNames[resource.Name] = true
	}

	// Check for cross-type name conflicts
	for k8sName := range k8sNames {
		if gcpNames[k8sName] {
			return fmt.Errorf("resource name '%s' is used in both K8s and GCP resources", k8sName)
		}
	}

	return nil
}

// validateFlowTimings validates that the sum of delays for each period doesn't exceed the period duration
func (v *FlowCustomValidator) validateFlowTimings(flow *kubecloudscalercloudv1alpha3.Flow) error {
	for i := range flow.Spec.Periods {
		period := &flow.Spec.Periods[i]
		// Find flows for this period
		var periodFlows []kubecloudscalercloudv1alpha3.Flows
		for _, f := range flow.Spec.Flows {
			if f.PeriodName == ptr.Deref(period.Name, "") {
				periodFlows = append(periodFlows, f)
			}
		}

		if len(periodFlows) == 0 {
			continue
		}

		// Get period duration
		periodDuration, err := v.getPeriodDuration(period)
		if err != nil {
			return fmt.Errorf("failed to get period duration for %s: %w", ptr.Deref(period.Name, ""), err)
		}

		// Calculate total delay for this period
		for _, f := range periodFlows {
			totalDelay := time.Duration(0)

			for _, resource := range f.Resources {
				if resource.Delay != nil {
					delay, err := time.ParseDuration(*resource.Delay)
					if err != nil {
						return fmt.Errorf("invalid delay format for resource %s: %w", resource.Name, err)
					}
					totalDelay += delay
				}
			}

			// Check if total delay exceeds period duration
			if totalDelay > periodDuration {
				return fmt.Errorf("total delay %v for period %s exceeds period duration %v",
					totalDelay, ptr.Deref(period.Name, ""), periodDuration)
			}
		}
	}

	return nil
}

// getPeriodDuration calculates the duration of a period
func (v *FlowCustomValidator) getPeriodDuration(period *common.ScalerPeriod) (time.Duration, error) {
	if period.Time.Recurring != nil {
		startTime, err := time.Parse("15:04", period.Time.Recurring.StartTime)
		if err != nil {
			return 0, fmt.Errorf("failed to parse start time: %w", err)
		}
		endTime, err := time.Parse("15:04", period.Time.Recurring.EndTime)
		if err != nil {
			return 0, fmt.Errorf("failed to parse end time: %w", err)
		}

		// Handle case where end time is next day
		if endTime.Before(startTime) {
			return 0, fmt.Errorf("end time is before start time")
		}

		return endTime.Sub(startTime), nil
	}

	if period.Time.Fixed != nil {
		startTime, err := time.Parse("2006-01-02 15:04:05", period.Time.Fixed.StartTime)
		if err != nil {
			return 0, fmt.Errorf("failed to parse start time: %w", err)
		}
		endTime, err := time.Parse("2006-01-02 15:04:05", period.Time.Fixed.EndTime)
		if err != nil {
			return 0, fmt.Errorf("failed to parse end time: %w", err)
		}

		// Handle case where end time is next day
		if endTime.Before(startTime) {
			return 0, fmt.Errorf("end time is before start time")
		}

		return endTime.Sub(startTime), nil
	}

	return 0, fmt.Errorf("no valid time period found")
}
