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

package utils

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReconcileResult represents the result of a reconciliation step.
type ReconcileResult struct {
	// ShouldStop indicates if reconciliation should stop.
	ShouldStop bool
	// RequeueAfter indicates when to requeue (0 means no requeue).
	RequeueAfter time.Duration
	// Error is any error that occurred.
	Error error
}

// HandleFinalizerReconcile handles finalizer management and returns a reconcile result.
// This is a convenience wrapper around HandleFinalizer for use in reconcile loops.
func HandleFinalizerReconcile(
	ctx context.Context,
	k8sClient client.Client,
	obj client.Object,
	finalizer string,
	logger *zerolog.Logger,
) ReconcileResult {
	result, err := HandleFinalizer(ctx, k8sClient, obj, finalizer, logger)
	if err != nil {
		return ReconcileResult{
			ShouldStop: true,
			Error:      err,
		}
	}

	if result.ShouldStop {
		return ReconcileResult{ShouldStop: true}
	}

	// If object is being deleted, remove finalizer and stop reconciliation
	if result.IsDeleting {
		if err := RemoveFinalizer(ctx, k8sClient, obj, finalizer, logger); err != nil {
			return ReconcileResult{
				ShouldStop: true,
				Error:      err,
			}
		}
		return ReconcileResult{ShouldStop: true}
	}

	return ReconcileResult{ShouldStop: false}
}

// HandleRunOncePeriod handles run-once period errors and returns appropriate requeue duration.
func HandleRunOncePeriod(periodEndTime time.Time, delaySeconds int) ctrl.Result {
	return ctrl.Result{
		RequeueAfter: time.Until(periodEndTime.Add(time.Duration(delaySeconds) * time.Second)),
	}
}

// ShouldSkipNoActionPeriod checks if reconciliation should be skipped for noaction period.
func ShouldSkipNoActionPeriod(currentPeriodName, statusPeriodName string) bool {
	return currentPeriodName == "noaction" && statusPeriodName == "noaction"
}
