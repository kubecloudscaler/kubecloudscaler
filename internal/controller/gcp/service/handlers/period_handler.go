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
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// RequeueDelaySeconds is the delay in seconds before requeuing a run-once period.
	RequeueDelaySeconds = 5
)

// PeriodHandler validates and determines the current time period.
// This handler validates period configuration and sets the active period for scaling.
//
// Responsibilities:
//   - Validate period configuration
//   - Determine current active period
//   - Configure resource management settings
//   - Handle "no action" periods (skip remaining handlers)
//   - Handle run-once periods (requeue until period ends)
//
// Error Handling:
//   - Invalid period configuration: Critical error (stops chain)
//   - Run-once period: Requeue (not an error)
type PeriodHandler struct {
	next service.Handler
}

// NewPeriodHandler creates a new period validation handler.
func NewPeriodHandler() service.Handler {
	return &PeriodHandler{}
}

// Execute implements the Handler interface.
// It validates periods and determines the current active period.
func (h *PeriodHandler) Execute(req *service.ReconciliationContext) (ctrl.Result, error) {
	req.Logger.Debug().Msg("validating period configuration")

	scaler := req.Scaler

	// Configure resource management settings
	resourceConfig := resources.Config{
		GCP: &gcpUtils.Config{
			Client:            req.GCPClient,
			ProjectID:         scaler.Spec.Config.ProjectID,
			Region:            scaler.Spec.Config.Region,
			Names:             scaler.Spec.Resources.Names,
			LabelSelector:     scaler.Spec.Resources.LabelSelector,
			DefaultPeriodType: scaler.Spec.Config.DefaultPeriodType,
		},
	}

	// Convert []common.ScalerPeriod to []*common.ScalerPeriod for utils.ValidatePeriod
	periods := make([]*common.ScalerPeriod, len(scaler.Spec.Periods))
	for i := range scaler.Spec.Periods {
		periods[i] = &scaler.Spec.Periods[i]
	}

	// Capture previous period name before ValidatePeriod mutates the status in-place.
	// Same fix as the K8s controller: ValidatePeriod overwrites status.CurrentPeriod
	// immediately, so comparing scaler.Status.CurrentPeriod.Name after the call always
	// sees the new value, causing the noaction skip to fire on every transition.
	prevPeriodName := ""
	if scaler.Status.CurrentPeriod != nil {
		prevPeriodName = scaler.Status.CurrentPeriod.Name
	}

	// Validate and determine the current time period
	period, err := utils.ValidatePeriod(
		req.Logger,
		periods,
		&scaler.Status,
		scaler.Spec.Config.RestoreOnDelete && req.ShouldFinalize,
	)
	if err != nil {
		// Handle run-once period - requeue until the period ends
		if errors.Is(err, utils.ErrRunOncePeriod) {
			requeueAfter := time.Until(period.GetEndTime.Add(RequeueDelaySeconds * time.Second))
			req.Logger.Info().Dur("requeue_after", requeueAfter).Msg("run-once period active, requeuing")
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}

		// Invalid period configuration - critical error
		req.Logger.Error().Err(err).Msg("period validation failed")
		return ctrl.Result{}, service.NewCriticalError(err)
	}

	resourceConfig.GCP.Period = period
	req.Period = period
	req.ResourceConfig = resourceConfig

	// Skip reconciliation only when the controller was already in "noaction" on the previous
	// cycle. If we just transitioned from an active period the scaling handler must still run
	// to restore resource state.
	if prevPeriodName == "noaction" && period.Name == "noaction" {
		req.Logger.Info().Msg("no action period detected, skipping reconciliation")
		req.SkipRemaining = true
		return ctrl.Result{RequeueAfter: utils.ReconcileSuccessDuration}, nil
	}

	req.Logger.Info().Str("period", period.Name).Msg("period validated successfully")
	if h.next != nil {
		return h.next.Execute(req)
	}
	return ctrl.Result{}, nil
}

// SetNext sets the next handler in the chain.
func (h *PeriodHandler) SetNext(next service.Handler) {
	h.next = next
}
