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
	ctrl "sigs.k8s.io/controller-runtime"
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
func (h *StatusHandler) Execute(req *service.ReconciliationContext) (ctrl.Result, error) {
	req.Logger.Debug().Msg("updating scaler status")

	scaler := req.Scaler

	ctx := req.Ctx

	// Handle finalizer cleanup if deletion is in progress
	if req.ShouldFinalize {
		req.Logger.Info().Msg("removing finalizer for deleted scaler")
		controllerutil.RemoveFinalizer(scaler, ScalerFinalizer)
		if err := req.Client.Update(ctx, scaler); err != nil {
			req.Logger.Error().Err(err).Msg("failed to remove finalizer")
			return ctrl.Result{RequeueAfter: transientRequeueAfter}, nil
		}
		// Finalizer removed successfully, stop chain
		return ctrl.Result{}, nil
	}

	// Build the desired status from the in-memory state set by PeriodHandler and ScalingHandler.
	// This must be snapshotted before the retry loop because re-fetching the object from the
	// cluster would overwrite fields like Spec/SpecSHA/Type/Name that PeriodHandler wrote.
	if scaler.Status.CurrentPeriod == nil {
		scaler.Status.CurrentPeriod = &common.ScalerStatusPeriod{}
	}
	scaler.Status.CurrentPeriod.Successful = req.SuccessResults
	scaler.Status.CurrentPeriod.Failed = req.FailedResults
	scaler.Status.Comments = ptr.To("time period processed")

	desiredPeriod := *scaler.Status.CurrentPeriod
	desiredComments := scaler.Status.Comments

	// Persist status updates to the cluster, retrying on conflict by re-fetching the latest version
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := req.Client.Get(ctx, req.Request.NamespacedName, scaler); err != nil {
			return err
		}
		// Restore desired status onto the freshly-fetched object (preserves resourceVersion)
		periodCopy := desiredPeriod
		scaler.Status.CurrentPeriod = &periodCopy
		scaler.Status.Comments = desiredComments
		return req.Client.Status().Update(ctx, scaler)
	}); err != nil {
		req.Logger.Error().Err(err).Msg("unable to update scaler status")
		return ctrl.Result{RequeueAfter: transientRequeueAfter}, nil
	}

	req.Logger.Info().
		Str("name", scaler.Name).
		Str("namespace", scaler.Namespace).
		Msg("scaler status updated successfully")

	// Requeue for the next reconciliation cycle
	return ctrl.Result{RequeueAfter: utils.ReconcileSuccessDuration}, nil
}

// SetNext sets the next handler in the chain.
func (h *StatusHandler) SetNext(next service.Handler) {
	h.next = next
}
