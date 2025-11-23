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

	"github.com/rs/zerolog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// FinalizerResult represents the result of finalizer handling.
type FinalizerResult struct {
	// ShouldStop indicates if reconciliation should stop.
	ShouldStop bool
	// IsDeleting indicates if the object is being deleted.
	IsDeleting bool
	// Requeue indicates if the operation should be requeued.
	Requeue bool
}

// HandleFinalizer manages finalizers for Kubernetes objects.
// It ensures proper cleanup operations can be performed before deletion.
//
// Parameters:
//   - ctx: The context for the operation
//   - k8sClient: The Kubernetes client for updates
//   - obj: The object to manage finalizers for (must implement client.Object)
//   - finalizer: The finalizer name to use
//   - logger: Logger for logging operations
//
// Returns:
//   - FinalizerResult: Contains information about the finalizer state
//   - error: Any error that occurred during the operation
func HandleFinalizer(
	ctx context.Context,
	k8sClient client.Client,
	obj client.Object,
	finalizer string,
	logger *zerolog.Logger,
) (FinalizerResult, error) {
	result := FinalizerResult{}

	// Check if the object is being deleted
	if obj.GetDeletionTimestamp().IsZero() {
		// Object is not being deleted - ensure finalizer is present
		if !controllerutil.ContainsFinalizer(obj, finalizer) {
			logger.Info().Msg("adding finalizer")
			controllerutil.AddFinalizer(obj, finalizer)
			if err := k8sClient.Update(ctx, obj); err != nil {
				return FinalizerResult{ShouldStop: true, Requeue: true}, err
			}
			return FinalizerResult{ShouldStop: true, Requeue: true}, nil
		}
		return FinalizerResult{ShouldStop: false}, nil
	}

	// Object is being deleted - handle finalizer cleanup
	if controllerutil.ContainsFinalizer(obj, finalizer) {
		logger.Info().Msg("deleting object with finalizer")
		result.IsDeleting = true
		return result, nil
	}

	// Finalizer already removed, stop reconciliation
	return FinalizerResult{ShouldStop: true}, nil
}

// RemoveFinalizer removes the finalizer from an object and updates it.
//
// Parameters:
//   - ctx: The context for the operation
//   - k8sClient: The Kubernetes client for updates
//   - obj: The object to remove the finalizer from (must implement client.Object)
//   - finalizer: The finalizer name to remove
//   - logger: Logger for logging operations
//
// Returns:
//   - error: Any error that occurred during the operation
func RemoveFinalizer(
	ctx context.Context,
	k8sClient client.Client,
	obj client.Object,
	finalizer string,
	logger *zerolog.Logger,
) error {
	logger.Info().Msg("removing finalizer")
	controllerutil.RemoveFinalizer(obj, finalizer)
	if err := k8sClient.Update(ctx, obj); err != nil {
		return err
	}
	return nil
}

// DefaultFinalizerName returns the default finalizer name for scalers.
func DefaultFinalizerName() string {
	return "kubecloudscaler.cloud/finalizer"
}
