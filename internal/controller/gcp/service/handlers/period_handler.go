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

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/metrics"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

const (
	// RequeueDelaySeconds is the delay in seconds before requeuing a run-once period.
	RequeueDelaySeconds = 5
)

// PeriodHandler validates and determines the current time period.
type PeriodHandler struct {
	next service.Handler
}

// NewPeriodHandler creates a new period validation handler.
func NewPeriodHandler() service.Handler {
	return &PeriodHandler{}
}

// Execute validates periods and determines the current active period.
func (h *PeriodHandler) Execute(ctx *service.ReconciliationContext) error {
	scaler := ctx.Scaler

	ctx.ResourceConfig = resources.Config{
		GCP: &gcpUtils.Config{
			Client:            ctx.GCPClient,
			ProjectID:         scaler.Spec.Config.ProjectID,
			Region:            scaler.Spec.Config.Region,
			Names:             scaler.Spec.Resources.Names,
			LabelSelector:     scaler.Spec.Resources.LabelSelector,
			DefaultPeriodType: scaler.Spec.Config.DefaultPeriodType,
		},
	}

	periods := make([]*common.ScalerPeriod, len(scaler.Spec.Periods))
	for i := range scaler.Spec.Periods {
		periods[i] = &scaler.Spec.Periods[i]
	}

	prevPeriodName := ""
	if scaler.Status.CurrentPeriod != nil {
		prevPeriodName = scaler.Status.CurrentPeriod.Name
	}

	period, err := utils.SetActivePeriod(
		ctx.Ctx,
		ctx.Logger,
		periods,
		&scaler.Status,
		scaler.Spec.Config.RestoreOnDelete && ctx.ShouldFinalize,
	)
	if err != nil {
		if errors.Is(err, utils.ErrRunOncePeriod) {
			metrics.PeriodEvaluationTotal.WithLabelValues(metrics.ControllerGCP, metrics.OutcomeRunOnceSkip).Inc()
			requeueAfter := time.Until(period.GetEndTime.Add(RequeueDelaySeconds * time.Second))
			if ctx.RequeueAfter == 0 {
				ctx.RequeueAfter = requeueAfter
			}
			ctx.Logger.Info().Dur("requeue_after", requeueAfter).Msg("run-once period active")
			ctx.SkipRemaining = true
			return nil
		}

		metrics.PeriodEvaluationTotal.WithLabelValues(metrics.ControllerGCP, metrics.OutcomeError).Inc()
		ctx.Logger.Error().Err(err).Msg("period validation error")
		return service.NewCriticalError(err)
	}

	ctx.ResourceConfig.GCP.Period = period
	ctx.Period = period

	if prevPeriodName == "noaction" && period.Name == "noaction" {
		metrics.PeriodEvaluationTotal.WithLabelValues(metrics.ControllerGCP, metrics.OutcomeNoaction).Inc()
		ctx.Logger.Info().Msg("noaction period, skipping")
		ctx.SkipRemaining = true
		if ctx.RequeueAfter == 0 {
			ctx.RequeueAfter = utils.ReconcileSuccessDuration
		}
		return nil
	}

	metrics.PeriodEvaluationTotal.WithLabelValues(metrics.ControllerGCP, metrics.OutcomeActive).Inc()
	ctx.Logger.Info().Str("period", period.Name).Str("type", period.Type).Msg("period active")
	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext sets the next handler in the chain.
func (h *PeriodHandler) SetNext(next service.Handler) {
	h.next = next
}
