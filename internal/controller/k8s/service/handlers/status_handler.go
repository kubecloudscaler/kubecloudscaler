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
			ctx.Logger.Error().Err(err).Msg("failed to remove finalizer")
			ctx.RequeueAfter = utils.ReconcileErrorDuration
			return service.NewRecoverableError(err)
		}
		controllerutil.RemoveFinalizer(ctx.Scaler, ScalerFinalizer)
		// Finalizer removed successfully, no requeue needed
		ctx.RequeueAfter = 0
		return nil
	}

	// Build the desired status from the in-memory state set by PeriodHandler and ScalingHandler.
	// This must be snapshotted before the retry loop because re-fetching the object from the
	// cluster would overwrite fields like Spec/SpecSHA/Type/Name that PeriodHandler wrote.
	if ctx.Scaler.Status.CurrentPeriod == nil {
		ctx.Scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{}
	}
	ctx.Scaler.Status.CurrentPeriod.Successful = ctx.SuccessResults
	ctx.Scaler.Status.CurrentPeriod.Failed = ctx.FailedResults
	ctx.Scaler.Status.Comments = ptr.To("time period processed")

	desiredPeriod := *ctx.Scaler.Status.CurrentPeriod
	desiredComments := ctx.Scaler.Status.Comments

	// Persist status updates to the cluster, retrying on conflict by re-fetching the latest version
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, ctx.Scaler); err != nil {
			return err
		}
		// Restore desired status onto the freshly-fetched object (preserves resourceVersion)
		periodCopy := desiredPeriod
		ctx.Scaler.Status.CurrentPeriod = &periodCopy
		ctx.Scaler.Status.Comments = desiredComments
		return ctx.Client.Status().Update(ctx.Ctx, ctx.Scaler)
	}); err != nil {
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
