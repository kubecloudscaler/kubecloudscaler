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

	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// ScalerFinalizer is the finalizer name for GCP scaler resources.
	ScalerFinalizer = "kubecloudscaler.cloud/finalizer"
)

// FinalizerHandler manages finalizer lifecycle (add/remove) for the scaler resource.
// This handler ensures proper cleanup when the resource is deleted.
//
// Responsibilities:
//   - Add finalizer if object is not being deleted
//   - Set ShouldFinalize flag if object is being deleted with finalizer
//   - Skip remaining handlers if finalizer already removed
//
// Error Handling:
//   - Update failures: Recoverable error (allows retry with requeue)
type FinalizerHandler struct {
	next service.Handler
}

// NewFinalizerHandler creates a new finalizer handler.
func NewFinalizerHandler() service.Handler {
	return &FinalizerHandler{}
}

// Execute implements the Handler interface.
// It manages the finalizer lifecycle for the scaler resource.
func (h *FinalizerHandler) Execute(ctx *service.ReconciliationContext) error {
	ctx.Logger.Debug().Msg("managing finalizer")

	scaler := ctx.Scaler

	// Check if the object is being deleted by examining the DeletionTimestamp
	if scaler.DeletionTimestamp.IsZero() {
		// Object is not being deleted - ensure finalizer is present
		if !controllerutil.ContainsFinalizer(scaler, ScalerFinalizer) {
			ctx.Logger.Info().Msg("adding finalizer")
			controllerutil.AddFinalizer(scaler, ScalerFinalizer)
			if err := ctx.Client.Update(ctx.Ctx, scaler); err != nil {
				ctx.Logger.Error().Err(err).Msg("failed to add finalizer")
				ctx.RequeueAfter = transientRequeueAfter
				return service.NewRecoverableError(fmt.Errorf("add finalizer: %w", err))
			}
		}
		// Finalizer present or added successfully, continue chain
		if h.next != nil && !ctx.SkipRemaining {
			return h.next.Execute(ctx)
		}
		return nil
	}

	// Object is being deleted
	if controllerutil.ContainsFinalizer(scaler, ScalerFinalizer) {
		// Finalizer present - set flag for cleanup and continue
		ctx.Logger.Info().Msg("scaler being deleted with finalizer, preparing for cleanup")
		ctx.ShouldFinalize = true
		if h.next != nil && !ctx.SkipRemaining {
			return h.next.Execute(ctx)
		}
		return nil
	}

	// Finalizer already removed - skip remaining handlers
	ctx.Logger.Info().Msg("finalizer already removed, skipping reconciliation")
	ctx.SkipRemaining = true
	return nil
}

// SetNext sets the next handler in the chain.
func (h *FinalizerHandler) SetNext(next service.Handler) {
	h.next = next
}
