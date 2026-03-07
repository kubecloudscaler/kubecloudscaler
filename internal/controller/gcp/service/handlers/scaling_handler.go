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
	"fmt"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

// ScalingHandler scales GCP resources based on the current period.
// This handler performs the actual resource scaling operations.
//
// Responsibilities:
//   - Validate and filter resource types
//   - Scale each resource type according to period configuration
//   - Collect success and failure results
//   - Continue chain even if individual resources fail (tracked in FailedResults)
//
// Error Handling:
//   - Individual resource failures: Continue chain, track in FailedResults
//   - Critical failures: Should be rare, most scaling errors are recoverable
type ScalingHandler struct {
	next service.Handler
}

// NewScalingHandler creates a new resource scaling handler.
func NewScalingHandler() service.Handler {
	return &ScalingHandler{}
}

// Execute implements the Handler interface.
// It scales GCP resources based on the current period configuration.
func (h *ScalingHandler) Execute(ctx *service.ReconciliationContext) error {
	var (
		successResults []common.ScalerStatusSuccess
		failedResults  []common.ScalerStatusFailed
	)

	resourceList := validResourceList(ctx.Scaler)

	// Process each resource type and perform scaling operations
	for _, resource := range resourceList {
		// Create a resource handler for the specific resource type
		curResource, err := resources.NewResource(ctx.Ctx, resource, ctx.ResourceConfig, ctx.Logger)
		if err != nil {
			ctx.Logger.Error().Err(err).Str("resource", resource).Msg("unable to get resource handler")
			failedResults = append(failedResults, common.ScalerStatusFailed{
				Kind:   resource,
				Name:   resource,
				Reason: fmt.Sprintf("unable to get resource handler: %v", err),
			})
			continue
		}

		// Execute the scaling operation for this resource type
		success, failed, err := curResource.SetState(ctx.Ctx)
		if err != nil {
			ctx.Logger.Error().Err(err).Str("resource", resource).Msg("unable to set resource state")
			failedResults = append(failedResults, common.ScalerStatusFailed{
				Kind:   resource,
				Name:   resource,
				Reason: fmt.Sprintf("unable to set resource state: %v", err),
			})
			continue
		}

		// Collect results for status reporting
		successResults = append(successResults, success...)
		failedResults = append(failedResults, failed...)
	}

	ctx.SuccessResults = successResults
	ctx.FailedResults = failedResults

	ctx.Logger.Debug().
		Int("success", len(successResults)).
		Int("failed", len(failedResults)).
		Strs("resources", resourceList).
		Msg("scaling done")

	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext sets the next handler in the chain.
func (h *ScalingHandler) SetNext(next service.Handler) {
	h.next = next
}

// validResourceList validates and filters the list of resources to be scaled.
// It ensures that only valid resource types are included for GCP resources.
func validResourceList(scaler *kubecloudscalerv1alpha3.Gcp) []string {
	// Default to compute instances if no resources are specified
	if len(scaler.Spec.Resources.Types) == 0 {
		return []string{resources.DefaultGCPResourceType}
	}
	return scaler.Spec.Resources.Types
}
