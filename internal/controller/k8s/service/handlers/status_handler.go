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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
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
	// Handle finalizer cleanup if the object is being deleted
	if ctx.ShouldFinalize {
		ctx.Logger.Info().Str("name", ctx.Scaler.Name).Msg("removing finalizer")
		if err := patchRemoveFinalizer(ctx); err != nil {
			if apierrors.IsNotFound(err) {
				// Scaler has already been fully deleted — cleanup is done.
				ctx.Logger.Debug().Msg("scaler already gone, finalizer cleanup is a no-op")
				ctx.RequeueAfter = 0
				return nil
			}
			ctx.Logger.Error().Err(err).Msg("failed to remove finalizer")
			ctx.RequeueAfter = utils.ReconcileErrorDuration
			return service.NewRecoverableError(err)
		}
		controllerutil.RemoveFinalizer(ctx.Scaler, ScalerFinalizer)
		// Finalizer removed successfully, no requeue needed
		ctx.RequeueAfter = 0
		return nil
	}

	// Capture desired status from in-memory state set by earlier handlers. Taken as a
	// DeepCopy so the retry loop's fresh Get cannot alias the Successful/Failed slices.
	if ctx.Scaler.Status.CurrentPeriod == nil {
		ctx.Scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{}
	}
	ctx.Scaler.Status.CurrentPeriod.Successful = ctx.SuccessResults
	ctx.Scaler.Status.CurrentPeriod.Failed = ctx.FailedResults
	ctx.Scaler.Status.Comments = ptr.To("time period processed")

	desiredPeriod := ctx.Scaler.Status.CurrentPeriod.DeepCopy()
	desiredComments := ctx.Scaler.Status.Comments

	// Persist status via optimistic-locked merge patch, scoped to the status subresource.
	// Patching (not Update) transmits only the fields we changed and respects
	// resourceVersion on conflict. Retry re-fetches the latest object inside the closure so
	// each attempt builds its patch base from fresh state.
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &kubecloudscalerv1alpha3.K8s{}
		if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, latest); err != nil {
			return err
		}
		patch := client.MergeFromWithOptions(latest.DeepCopy(), client.MergeFromWithOptimisticLock{})
		latest.Status.CurrentPeriod = desiredPeriod.DeepCopy()
		latest.Status.Comments = desiredComments
		return ctx.Client.Status().Patch(ctx.Ctx, latest, patch)
	}); err != nil {
		if apierrors.IsNotFound(err) {
			// Scaler was deleted mid-reconcile — nothing to update.
			ctx.Logger.Debug().Msg("scaler vanished before status could be updated, skipping")
			return nil
		}
		ctx.Logger.Error().Err(err).Msg("unable to update scaler status")
		ctx.RequeueAfter = utils.ReconcileErrorDuration
		return service.NewRecoverableError(err)
	}

	// Single summary log per successful reconciliation (period + counts)
	logEvent := ctx.Logger.Info().
		Str("name", ctx.Scaler.Name).
		Int("success", len(ctx.SuccessResults)).
		Int("failed", len(ctx.FailedResults))
	if ctx.Period != nil {
		logEvent = logEvent.Str("period", ctx.Period.Name)
	}
	logEvent.Msg("reconciled")

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

// patchRemoveFinalizer removes ScalerFinalizer via an optimistic-locked merge patch, re-fetching
// and retrying on 409 conflicts. Scoped to metadata.finalizers so neither spec nor status is
// transmitted. No-op if the finalizer is already absent.
func patchRemoveFinalizer(ctx *service.ReconciliationContext) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &kubecloudscalerv1alpha3.K8s{}
		if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, latest); err != nil {
			return err
		}
		if !controllerutil.ContainsFinalizer(latest, ScalerFinalizer) {
			return nil
		}
		patch := client.MergeFromWithOptions(latest.DeepCopy(), client.MergeFromWithOptimisticLock{})
		controllerutil.RemoveFinalizer(latest, ScalerFinalizer)
		return ctx.Client.Patch(ctx.Ctx, latest, patch)
	})
}
