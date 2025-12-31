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
	"errors"
	"time"

	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	k8sUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

// RequeueDelaySeconds is the delay in seconds before requeuing a run-once period.
const RequeueDelaySeconds = 5

// PeriodHandler is a handler that validates and determines the current time period for scaling operations.
type PeriodHandler struct {
	next service.Handler
}

// NewPeriodHandler creates a new period validation handler.
func NewPeriodHandler() service.Handler {
	return &PeriodHandler{}
}

// Execute validates and determines the current time period and adds it to the reconciliation context.
//
// Behavior:
//   - Configures resource management settings
//   - Validates time periods and determines current period
//   - If run-once period: Sets RequeueAfter, stops chain
//   - If "noaction" period matches current status: Sets SkipRemaining, stops chain
func (h *PeriodHandler) Execute(ctx *service.ReconciliationContext) error {
	ctx.Logger.Debug().Msg("validating and determining current period")

	// Configure resource management settings
	ctx.ResourceConfig = resources.Config{
		K8s: &k8sUtils.Config{
			Client:                       ctx.K8sClient,
			DynamicClient:                ctx.DynamicClient,
			Names:                        ctx.Scaler.Spec.Resources.Names,
			Namespaces:                   ctx.Scaler.Spec.Config.Namespaces,
			ExcludeNamespaces:            ctx.Scaler.Spec.Config.ExcludeNamespaces,
			LabelSelector:                ctx.Scaler.Spec.Resources.LabelSelector,
			ForceExcludeSystemNamespaces: ctx.Scaler.Spec.Config.ForceExcludeSystemNamespaces,
		},
	}

	// Convert []common.ScalerPeriod to []*common.ScalerPeriod for utils.ValidatePeriod
	periods := make([]*common.ScalerPeriod, len(ctx.Scaler.Spec.Periods))
	for i := range ctx.Scaler.Spec.Periods {
		periods[i] = &ctx.Scaler.Spec.Periods[i]
	}

	// Validate and determine the current time period for scaling operations
	period, err := utils.ValidatePeriod(
		periods,
		&ctx.Scaler.Status,
		ctx.Scaler.Spec.Config.RestoreOnDelete && ctx.ShouldFinalize,
	)
	if err != nil {
		if errors.Is(err, utils.ErrRunOncePeriod) {
			ctx.Logger.Info().Msg("run-once period detected, requeuing until period ends")
			if ctx.RequeueAfter == 0 {
				ctx.RequeueAfter = time.Until(period.GetEndTime.Add(RequeueDelaySeconds * time.Second))
			}
			ctx.SkipRemaining = true
			return nil
		}

		ctx.Logger.Error().Err(err).Msg("unable to validate period")
		ctx.Scaler.Status.Comments = ptr.To(err.Error())
		return service.NewCriticalError(err)
	}
	ctx.Period = period
	ctx.ResourceConfig.K8s.Period = period

	// Check for "noaction" period
	if ctx.Scaler.Status.CurrentPeriod != nil &&
		ctx.Period.Name == "noaction" &&
		ctx.Scaler.Status.CurrentPeriod.Name == ctx.Period.Name {
		ctx.Logger.Debug().Msg("no action period, skipping reconciliation")
		ctx.SkipRemaining = true
		if ctx.RequeueAfter == 0 {
			ctx.RequeueAfter = utils.ReconcileSuccessDuration
		}
		return nil
	}

	ctx.Logger.Info().Str("period", ctx.Period.Name).Str("type", ctx.Period.Type).Msg("period determined")

	// Call next handler in chain
	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext establishes the next handler in the chain.
func (h *PeriodHandler) SetNext(next service.Handler) {
	h.next = next
}
