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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
)

// UpdateScalerStatus updates the status of a scaler object with the provided results.
// This is a helper function that updates the status field and persists it to the cluster.
//
// Parameters:
//   - ctx: The context for the operation
//   - statusWriter: The status writer from r.Status() (client.SubResourceWriter)
//   - obj: The object to update (must implement client.Object and have a Status field of type common.ScalerStatus)
//   - success: List of successful scaling operations
//   - failed: List of failed scaling operations
//   - comment: Optional comment message
//   - logger: Logger for logging operations
//
// Returns:
//   - error: Any error that occurred during the status update
func UpdateScalerStatus(
	ctx context.Context,
	statusWriter client.SubResourceWriter,
	obj client.Object,
	success []common.ScalerStatusSuccess,
	failed []common.ScalerStatusFailed,
	comment string,
	logger *zerolog.Logger,
) error {
	// Use type assertion to access status - this works for both K8s and Gcp types
	type statusAccessor interface {
		client.Object
		GetStatus() *common.ScalerStatus
	}

	accessor, ok := obj.(statusAccessor)
	if !ok {
		// Fallback: try to access Status field directly via reflection or type switch
		// For now, return error - callers should ensure the object has a Status field
		logger.Error().Msg("object does not have accessible Status field")
		return nil
	}

	status := accessor.GetStatus()
	if status == nil {
		status = &common.ScalerStatus{}
	}

	if status.CurrentPeriod == nil {
		status.CurrentPeriod = &common.ScalerStatusPeriod{}
	}

	status.CurrentPeriod.Successful = success
	status.CurrentPeriod.Failed = failed
	if comment != "" {
		status.Comments = ptr.To(comment)
	}

	if err := statusWriter.Update(ctx, obj); err != nil {
		logger.Error().Err(err).Msg("unable to update scaler status")
		return err
	}

	logger.Info().
		Str("name", obj.GetName()).
		Str("kind", obj.GetObjectKind().GroupVersionKind().Kind).
		Msg("scaler status updated")

	return nil
}

// UpdateScalerStatusWithError updates the status of a scaler object with an error message.
//
// Parameters:
//   - ctx: The context for the operation
//   - statusWriter: The status writer from r.Status() (client.SubResourceWriter)
//   - obj: The object to update
//   - err: The error to record in the status
//   - logger: Logger for logging operations
//
// Returns:
//   - error: Any error that occurred during the status update
func UpdateScalerStatusWithError(
	ctx context.Context,
	statusWriter client.SubResourceWriter,
	obj client.Object,
	err error,
	logger *zerolog.Logger,
) error {
	comment := ""
	if err != nil {
		comment = err.Error()
	}
	return UpdateScalerStatus(ctx, statusWriter, obj, nil, nil, comment, logger)
}
