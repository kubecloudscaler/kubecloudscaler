// Package utils provides utility functions for GCP resource management in the kubecloudscaler project.
package utils

import (
	"context"
	"fmt"
	"time"

	computepb "cloud.google.com/go/compute/apiv1/computepb"
)

// WaitForZoneOperation polls ZoneOperations until the operation completes or the context is cancelled.
func WaitForZoneOperation(ctx context.Context, clients *ClientSet, projectID, operationName, zone string) error {
	if clients == nil {
		return fmt.Errorf("clients cannot be nil")
	}
	if clients.ZoneOperations == nil {
		return fmt.Errorf("ZoneOperations client cannot be nil")
	}

	timer := time.NewTimer(OperationTimeoutMinutes * time.Minute)
	defer timer.Stop()
	ticker := time.NewTicker(OperationCheckIntervalSeconds * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("operation timed out")
		case <-ticker.C:
			op, err := clients.ZoneOperations.Get(ctx, &computepb.GetZoneOperationRequest{
				Operation: operationName,
				Project:   projectID,
				Zone:      zone,
			})
			if err != nil {
				return fmt.Errorf("failed to get operation status: %w", err)
			}
			if op.GetStatus() == computepb.Operation_DONE {
				if op.Error != nil && len(op.Error.Errors) > 0 {
					return fmt.Errorf("operation failed: %v", op.Error.Errors)
				}
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
