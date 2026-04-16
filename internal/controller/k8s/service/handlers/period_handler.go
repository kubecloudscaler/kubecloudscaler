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
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
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
//   - If run-once period (and not finalizing): Sets RequeueAfter, stops chain
//   - If "noaction" period matches current status (and not finalizing): Sets SkipRemaining, stops chain
//   - During deletion (ShouldFinalize), never skips: StatusHandler must run to remove the finalizer
func (h *PeriodHandler) Execute(ctx *service.ReconciliationContext) error {
	h.configureResourceSettings(ctx)

	prevPeriodName := previousPeriodName(ctx.Scaler.Status.CurrentPeriod)

	period, err := h.resolveActivePeriod(ctx)
	if err != nil {
		return err
	}
	ctx.Period = period
	ctx.ResourceConfig.K8s.Period = period

	if h.shouldSkipNoaction(ctx, prevPeriodName) {
		ctx.Logger.Debug().Str("period", periodPkg.NoactionPeriodName).Msg("no action period, skipping")
		ctx.SkipRemaining = true
		if ctx.RequeueAfter == 0 {
			ctx.RequeueAfter = utils.ReconcileSuccessDuration
		}
		return nil
	}

	ctx.Logger.Info().Str("period", ctx.Period.Name).Str("type", string(ctx.Period.Type)).Msg("active period set")

	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

func (h *PeriodHandler) configureResourceSettings(ctx *service.ReconciliationContext) {
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
}

func previousPeriodName(cp *common.ScalerStatusPeriod) string {
	if cp != nil {
		return cp.Name
	}
	return ""
}

// resolveActivePeriod determines the active period, handling the run-once early exit
// unless the resource is being deleted (ShouldFinalize).
func (h *PeriodHandler) resolveActivePeriod(ctx *service.ReconciliationContext) (*periodPkg.Period, error) {
	periods := make([]*common.ScalerPeriod, len(ctx.Scaler.Spec.Periods))
	for i := range ctx.Scaler.Spec.Periods {
		periods[i] = &ctx.Scaler.Spec.Periods[i]
	}

	period, err := utils.SetActivePeriod(
		ctx.Logger,
		periods,
		&ctx.Scaler.Status,
		ctx.Scaler.Spec.Config.RestoreOnDelete && ctx.ShouldFinalize,
	)
	if err != nil {
		if errors.Is(err, utils.ErrRunOncePeriod) && !ctx.ShouldFinalize {
			ctx.Logger.Info().Msg("run-once period detected, requeuing until period ends")
			if ctx.RequeueAfter == 0 {
				ctx.RequeueAfter = time.Until(period.EndTime.Add(RequeueDelaySeconds * time.Second))
			}
			ctx.SkipRemaining = true
			return period, nil
		}

		ctx.Logger.Error().Err(err).Msg("unable to validate period")
		ctx.Scaler.Status.Comments = ptr.To(err.Error())
		return nil, service.NewCriticalError(err)
	}
	return period, nil
}

// shouldSkipNoaction returns true when we can safely skip the rest of the chain
// because the period is still "noaction" (steady state). During deletion this
// must always return false so that StatusHandler can remove the finalizer.
func (h *PeriodHandler) shouldSkipNoaction(ctx *service.ReconciliationContext, prevPeriodName string) bool {
	if ctx.ShouldFinalize {
		return false
	}
	return prevPeriodName == periodPkg.NoactionPeriodName && ctx.Period.Name == periodPkg.NoactionPeriodName
}

// SetNext establishes the next handler in the chain.
func (h *PeriodHandler) SetNext(next service.Handler) {
	h.next = next
}
