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

package handlers

import (
	"slices"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/metrics"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

// ScalingHandler is a handler that scales K8s resources based on the determined period.
type ScalingHandler struct {
	next service.Handler
}

// NewScalingHandler creates a new ScalingHandler.
func NewScalingHandler() service.Handler {
	return &ScalingHandler{}
}

// Execute scales K8s resources and updates the reconciliation context with results.
//
// Behavior:
//   - Validates and filters resource list
//   - Processes each resource type for scaling
//   - Collects success and failure results
//   - Always continues to next handler (errors are collected, not returned)
func (h *ScalingHandler) Execute(ctx *service.ReconciliationContext) error {
	var (
		recSuccess []common.ScalerStatusSuccess
		recFailed  []common.ScalerStatusFailed
	)

	// Validate and filter the list of resources to be scaled
	resourceList, err := h.validResourceList(ctx)
	if err != nil {
		ctx.Logger.Error().Err(err).Msg("invalid resource list")
		recFailed = append(recFailed, common.ScalerStatusFailed{
			Kind:   "N/A",
			Name:   "N/A",
			Reason: err.Error(),
		})
		ctx.SuccessResults = recSuccess
		ctx.FailedResults = recFailed
		// Continue to next handler despite validation error
		if h.next != nil && !ctx.SkipRemaining {
			return h.next.Execute(ctx)
		}
		return nil
	}

	action := ""
	if ctx.Period != nil {
		action = ctx.Period.Type
	}

	// Process each resource type and perform scaling operations
	for _, resource := range resourceList {
		curResource, err := resources.NewResource(resource, ctx.ResourceConfig, ctx.Logger)
		if err != nil {
			ctx.Logger.Error().Err(err).Str("resource", resource).Msg("resource handler creation failed")
			metrics.ScalingOperationsTotal.WithLabelValues(metrics.ControllerK8s, resource, action, metrics.ResultFailure).Inc()
			recFailed = append(recFailed, common.ScalerStatusFailed{
				Kind:   resource,
				Name:   "N/A",
				Reason: err.Error(),
			})
			continue
		}

		success, failed, err := curResource.SetState(ctx.Ctx)
		if err != nil {
			ctx.Logger.Error().Err(err).Str("resource", resource).Msg("set state failed")
			metrics.ScalingOperationsTotal.WithLabelValues(metrics.ControllerK8s, resource, action, metrics.ResultFailure).Inc()
			recFailed = append(recFailed, common.ScalerStatusFailed{
				Kind:   resource,
				Name:   "N/A",
				Reason: err.Error(),
			})
			continue
		}

		if len(success) > 0 {
			metrics.ScalingOperationsTotal.WithLabelValues(
				metrics.ControllerK8s, resource, action, metrics.ResultSuccess,
			).Add(float64(len(success)))
		}
		if len(failed) > 0 {
			metrics.ScalingOperationsTotal.WithLabelValues(
				metrics.ControllerK8s, resource, action, metrics.ResultFailure,
			).Add(float64(len(failed)))
		}

		recSuccess = append(recSuccess, success...)
		recFailed = append(recFailed, failed...)
	}

	ctx.SuccessResults = recSuccess
	ctx.FailedResults = recFailed

	ctx.Logger.Info().
		Int("success_count", len(recSuccess)).
		Int("failed_count", len(recFailed)).
		Msg("scaling completed")

	// Call next handler in chain
	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext establishes the next handler in the chain.
func (h *ScalingHandler) SetNext(next service.Handler) {
	h.next = next
}

// validResourceList validates and filters the list of resources to be scaled.
// It ensures that only valid resource types are included and prevents mixing
// of application resources (deployments, statefulsets) with HPA resources.
func (h *ScalingHandler) validResourceList(ctx *service.ReconciliationContext) ([]string, error) {
	resourceTypes := ctx.Scaler.Spec.Resources.Types
	if len(resourceTypes) == 0 {
		resourceTypes = []string{resources.DefaultK8SResourceType}
	}

	output := make([]string, 0, len(resourceTypes))
	var (
		isApp bool // Flag indicating if app resources are present
		isHpa bool // Flag indicating if HPA resources are present
	)

	// Process each resource type and validate it
	for _, resource := range resourceTypes {
		// Check if this is an application resource (deployment, statefulset, etc.)
		if slices.Contains(utils.AppsResources, resource) {
			isApp = true
		}

		// Check if this is an HPA resource
		if slices.Contains(utils.HpaResources, resource) {
			isHpa = true
		}

		// Prevent mixing of app and HPA resources as they have different scaling behaviors
		if isHpa && isApp {
			ctx.Logger.Info().Str("reason", utils.ErrMixedAppsHPA.Error()).Msg("resource list invalid")
			return []string{}, utils.ErrMixedAppsHPA
		}

		// Add valid resource to the output list
		output = append(output, resource)
	}

	return output, nil
}
