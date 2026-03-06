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
	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/metrics"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

// ScalingHandler scales GCP resources based on the current period.
type ScalingHandler struct {
	next service.Handler
}

// NewScalingHandler creates a new resource scaling handler.
func NewScalingHandler() service.Handler {
	return &ScalingHandler{}
}

// Execute scales GCP resources based on the current period configuration.
func (h *ScalingHandler) Execute(ctx *service.ReconciliationContext) error {
	var (
		successResults []common.ScalerStatusSuccess
		failedResults  []common.ScalerStatusFailed
	)

	resourceList := validResourceList(ctx.Scaler)

	action := ""
	if ctx.Period != nil {
		action = ctx.Period.Type
	}

	for _, resource := range resourceList {
		curResource, err := resources.NewResource(resource, ctx.ResourceConfig, ctx.Logger)
		if err != nil {
			ctx.Logger.Error().Err(err).Str("resource", resource).Msg("resource handler creation failed")
			metrics.ScalingOperationsTotal.WithLabelValues(metrics.ControllerGCP, resource, action, metrics.ResultFailure).Inc()
			continue
		}

		success, failed, err := curResource.SetState(ctx.Ctx)
		if err != nil {
			ctx.Logger.Error().Err(err).Str("resource", resource).Msg("set state failed")
			metrics.ScalingOperationsTotal.WithLabelValues(metrics.ControllerGCP, resource, action, metrics.ResultFailure).Inc()
			continue
		}

		if len(success) > 0 {
			metrics.ScalingOperationsTotal.WithLabelValues(
				metrics.ControllerGCP, resource, action, metrics.ResultSuccess,
			).Add(float64(len(success)))
		}
		if len(failed) > 0 {
			metrics.ScalingOperationsTotal.WithLabelValues(
				metrics.ControllerGCP, resource, action, metrics.ResultFailure,
			).Add(float64(len(failed)))
		}

		successResults = append(successResults, success...)
		failedResults = append(failedResults, failed...)
	}

	ctx.SuccessResults = successResults
	ctx.FailedResults = failedResults

	ctx.Logger.Info().
		Int("success_count", len(successResults)).
		Int("failed_count", len(failedResults)).
		Msg("scaling completed")

	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext sets the next handler in the chain.
func (h *ScalingHandler) SetNext(next service.Handler) {
	h.next = next
}

func validResourceList(scaler *kubecloudscalerv1alpha3.Gcp) []string {
	if len(scaler.Spec.Resources.Types) == 0 {
		return []string{resources.DefaultGCPResourceType}
	}
	return scaler.Spec.Resources.Types
}
