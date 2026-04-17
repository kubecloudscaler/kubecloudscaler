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

	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

// StatusHandler updates the scaler status with operation results.
// This is the final handler in the chain and persists results to Kubernetes.
//
// Responsibilities:
//   - Handle finalizer cleanup if deletion is in progress
//   - Update scaler status with scaling results
//   - Persist status changes to Kubernetes
//   - Set requeue behavior for next reconciliation cycle
//
// Error Handling:
//   - Status update failures: Recoverable error (allows retry)
//   - Finalizer removal failures: Recoverable error (allows retry)
type StatusHandler struct {
	next service.Handler
}

// NewStatusHandler creates a new status update handler.
func NewStatusHandler() service.Handler {
	return &StatusHandler{}
}

// Execute implements the Handler interface.
// It updates the scaler status and handles finalizer cleanup.
func (h *StatusHandler) Execute(ctx *service.ReconciliationContext) error {
	scaler := ctx.Scaler

	if ctx.ShouldFinalize {
		ctx.Logger.Info().Str("name", scaler.Name).Msg("removing finalizer")
		if err := patchRemoveFinalizer(ctx); err != nil {
			ctx.Logger.Error().Err(err).Msg("failed to remove finalizer")
			ctx.RequeueAfter = transientRequeueAfter
			return service.NewRecoverableError(fmt.Errorf("remove finalizer: %w", err))
		}
		controllerutil.RemoveFinalizer(scaler, ScalerFinalizer)
		// Finalizer removed successfully, stop chain
		return nil
	}

	// Build the desired status from the in-memory state set by PeriodHandler and ScalingHandler.
	// Snapshot via DeepCopy before the retry loop: re-fetching overwrites fields like
	// Spec/SpecSHA/Type/Name that PeriodHandler wrote, and a shallow *scaler.Status.CurrentPeriod
	// would alias the `Spec *TimePeriod` pointer and the Successful/Failed slices.
	if scaler.Status.CurrentPeriod == nil {
		scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{}
	}
	scaler.Status.CurrentPeriod.Successful = ctx.SuccessResults
	scaler.Status.CurrentPeriod.Failed = ctx.FailedResults
	scaler.Status.Comments = ptr.To("time period processed")

	desiredPeriod := scaler.Status.CurrentPeriod.DeepCopy()
	desiredComments := scaler.Status.Comments

	// Persist status updates to the cluster, retrying on conflict by re-fetching the latest version
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := ctx.Client.Get(ctx.Ctx, ctx.Request.NamespacedName, scaler); err != nil {
			return err
		}
		// Restore desired status onto the freshly-fetched object (preserves resourceVersion).
		scaler.Status.CurrentPeriod = desiredPeriod.DeepCopy()
		scaler.Status.Comments = desiredComments
		return ctx.Client.Status().Update(ctx.Ctx, scaler)
	}); err != nil {
		ctx.Logger.Error().Err(err).Msg("unable to update scaler status")
		ctx.RequeueAfter = transientRequeueAfter
		return service.NewRecoverableError(fmt.Errorf("update scaler status: %w", err))
	}

	logEvent := ctx.Logger.Info().
		Str("name", scaler.Name).
		Int("success", len(ctx.SuccessResults)).
		Int("failed", len(ctx.FailedResults))
	if ctx.Period != nil {
		logEvent = logEvent.Str("period", ctx.Period.Name)
	}
	logEvent.Msg("reconciled")

	// Requeue for the next reconciliation cycle (first-write-wins, like K8s)
	if ctx.RequeueAfter == 0 {
		ctx.RequeueAfter = utils.ReconcileSuccessDuration
	}
	return nil
}

// SetNext sets the next handler in the chain.
func (h *StatusHandler) SetNext(next service.Handler) {
	h.next = next
}

// patchRemoveFinalizer removes ScalerFinalizer via an optimistic-locked merge patch, re-fetching
// and retrying on 409 conflicts. Scoped to metadata.finalizers so neither spec nor status is
// transmitted. No-op if the finalizer is already absent.
func patchRemoveFinalizer(ctx *service.ReconciliationContext) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest := &kubecloudscalerv1alpha3.Gcp{}
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
