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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

// ScalerFinalizer is the finalizer name used for K8s scaler resources.
const ScalerFinalizer = "kubecloudscaler.cloud/finalizer"

// FinalizerHandler is a handler that manages the finalizer lifecycle for the Scaler object.
type FinalizerHandler struct {
	next service.Handler
}

// NewFinalizerHandler creates a new FinalizerHandler.
func NewFinalizerHandler() service.Handler {
	return &FinalizerHandler{}
}

// Execute manages the finalizer lifecycle.
//
// Behavior:
//   - If not being deleted: Adds finalizer if not present, continues chain
//   - If being deleted with finalizer: Sets ShouldFinalize flag, continues chain
//   - If being deleted without finalizer: Sets SkipRemaining, stops chain
func (h *FinalizerHandler) Execute(ctx *service.ReconciliationContext) error {
	// Check if the object is being deleted
	if ctx.Scaler.DeletionTimestamp.IsZero() {
		// Object is not being deleted - ensure finalizer is present
		if !controllerutil.ContainsFinalizer(ctx.Scaler, ScalerFinalizer) {
			ctx.Logger.Info().Msg("adding finalizer")
			if err := patchAddFinalizer(ctx); err != nil {
				if apierrors.IsNotFound(err) {
					// Scaler was deleted between FetchHandler and this patch — nothing to do.
					ctx.Logger.Debug().Msg("scaler vanished before finalizer could be added, skipping")
					ctx.SkipRemaining = true
					return nil
				}
				ctx.Logger.Error().Err(err).Msg("failed to add finalizer")
				ctx.RequeueAfter = utils.ReconcileErrorDuration
				return service.NewRecoverableError(err)
			}
			controllerutil.AddFinalizer(ctx.Scaler, ScalerFinalizer)
		}
	} else {
		// Object is being deleted - handle finalizer cleanup
		if controllerutil.ContainsFinalizer(ctx.Scaler, ScalerFinalizer) {
			ctx.Logger.Info().Msg("deleting scaler with finalizer")
			ctx.ShouldFinalize = true // Signal subsequent handlers to perform cleanup
		} else {
			// Finalizer already removed, stop reconciliation
			ctx.Logger.Info().Msg("finalizer already removed, skipping remaining handlers")
			ctx.SkipRemaining = true // Signal the chain to stop
			return nil
		}
	}

	// Call next handler in chain
	if h.next != nil && !ctx.SkipRemaining {
		return h.next.Execute(ctx)
	}
	return nil
}

// SetNext establishes the next handler in the chain.
func (h *FinalizerHandler) SetNext(next service.Handler) {
	h.next = next
}

// patchAddFinalizer adds ScalerFinalizer via an optimistic-locked merge patch, re-fetching
// and retrying on 409 conflicts. Scoped to metadata.finalizers so neither spec nor status is
// transmitted. No-op if another reconcile added the finalizer concurrently.
func patchAddFinalizer(ctx *service.ReconciliationContext) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &kubecloudscalerv1alpha3.K8s{}
		if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, latest); err != nil {
			return err
		}
		if controllerutil.ContainsFinalizer(latest, ScalerFinalizer) {
			return nil
		}
		patch := client.MergeFromWithOptions(latest.DeepCopy(), client.MergeFromWithOptimisticLock{})
		controllerutil.AddFinalizer(latest, ScalerFinalizer)
		return ctx.Client.Patch(ctx.Ctx, latest, patch)
	})
}
