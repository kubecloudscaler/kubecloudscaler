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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

// StatusHandler is a handler that updates the scaler status with operation results.
// This is typically the last handler in the chain.
type StatusHandler struct {
	next service.Handler
}

// NewStatusHandler creates a new StatusHandler.
func NewStatusHandler() service.Handler {
	return &StatusHandler{}
}

// Execute updates the scaler status and performs finalizer cleanup if needed.
//
// Behavior:
//   - If ShouldFinalize: Removes finalizer, returns without requeue
//   - Otherwise: Updates status with success/failure results, sets requeue
func (h *StatusHandler) Execute(ctx *service.ReconciliationContext) error {
	ctx.Logger.Debug().Msg("updating scaler status")

	// Handle finalizer cleanup if the object is being deleted
	if ctx.ShouldFinalize {
		ctx.Logger.Info().Msg("removing finalizer")
		controllerutil.RemoveFinalizer(ctx.Scaler, ScalerFinalizer)
		if err := ctx.Client.Update(ctx.Ctx, ctx.Scaler); err != nil {
			ctx.Logger.Error().Err(err).Msg("failed to remove finalizer")
			ctx.RequeueAfter = utils.ReconcileErrorDuration
			return service.NewRecoverableError(err)
		}
		// Finalizer removed successfully, no requeue needed
		ctx.RequeueAfter = 0
		return nil
	}

	// Initialize CurrentPeriod if it's nil to prevent panics
	if ctx.Scaler.Status.CurrentPeriod == nil {
		ctx.Scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{}
	}

	// Update scaler status with operation results
	ctx.Scaler.Status.CurrentPeriod.Successful = ctx.SuccessResults
	ctx.Scaler.Status.CurrentPeriod.Failed = ctx.FailedResults
	ctx.Scaler.Status.Comments = ptr.To("time period processed")

	// Persist status updates to the cluster
	if err := ctx.Client.Status().Update(ctx.Ctx, ctx.Scaler); err != nil {
		ctx.Logger.Error().Err(err).Msg("unable to update scaler status")
		ctx.RequeueAfter = utils.ReconcileErrorDuration
		return service.NewRecoverableError(err)
	}

	ctx.Logger.Info().Str("name", ctx.Scaler.Name).Str("namespace", ctx.Scaler.Namespace).Msg("scaler status updated")

	// Set requeue for the next reconciliation cycle
	if ctx.RequeueAfter == 0 {
		ctx.RequeueAfter = utils.ReconcileSuccessDuration
	}

	// Call next handler in chain (if any)
	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext establishes the next handler in the chain.
func (h *StatusHandler) SetNext(next service.Handler) {
	h.next = next
}
