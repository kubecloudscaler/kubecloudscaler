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
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

// StatusHandler updates the scaler status with operation results.
// This is typically the last handler in the chain.
type StatusHandler struct {
	next service.Handler
}

// NewStatusHandler creates a new status update handler.
func NewStatusHandler() service.Handler {
	return &StatusHandler{}
}

// Execute updates the scaler status and performs finalizer cleanup if needed.
func (h *StatusHandler) Execute(ctx *service.ReconciliationContext) error {
	scaler := ctx.Scaler

	if ctx.ShouldFinalize {
		controllerutil.RemoveFinalizer(scaler, ScalerFinalizer)
		if err := ctx.Client.Update(ctx.Ctx, scaler); err != nil {
			ctx.Logger.Error().Err(err).Str("name", scaler.Name).Msg("remove finalizer failed")
			ctx.RequeueAfter = service.TransientRequeueAfter
			return nil
		}
		ctx.RequeueAfter = 0
		return nil
	}

	if scaler.Status.CurrentPeriod == nil {
		scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{}
	}
	scaler.Status.CurrentPeriod.Successful = ctx.SuccessResults
	scaler.Status.CurrentPeriod.Failed = ctx.FailedResults
	scaler.Status.Comments = ptr.To("time period processed")

	desiredPeriod := *scaler.Status.CurrentPeriod
	desiredComments := scaler.Status.Comments

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, scaler); err != nil {
			return err
		}
		periodCopy := desiredPeriod
		scaler.Status.CurrentPeriod = &periodCopy
		scaler.Status.Comments = desiredComments
		return ctx.Client.Status().Update(ctx.Ctx, scaler)
	}); err != nil {
		ctx.Logger.Error().Err(err).Str("name", scaler.Name).Msg("status update failed")
		ctx.RequeueAfter = service.TransientRequeueAfter
		return nil
	}

	ctx.Logger.Info().Str("name", scaler.Name).Str("namespace", scaler.Namespace).Msg("status updated")

	if ctx.RequeueAfter == 0 {
		ctx.RequeueAfter = utils.ReconcileSuccessDuration
	}

	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext sets the next handler in the chain.
func (h *StatusHandler) SetNext(next service.Handler) {
	h.next = next
}
