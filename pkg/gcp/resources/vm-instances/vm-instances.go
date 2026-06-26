// Package vminstances provides VM instance scaling functionality for GCP resources.
package vminstances

import (
	"context"
	"fmt"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
)

// SetState scales instances based on the current period
func (c *VMInstances) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	success := make([]common.ScalerStatusSuccess, 0)
	failed := make([]common.ScalerStatusFailed, 0)

	// Get all zones in the region
	zones, err := gcpUtils.GetZonesFromRegion(ctx, c.Config.Client, c.Config.ProjectID, c.Config.Region)
	if err != nil {
		return success, failed, fmt.Errorf("failed to get zones: %w", err)
	}

	// Get instances in the zones filtered by label selector (filter applied at GCP API level)
	filteredInstances, err := gcpUtils.GetInstancesInZones(ctx, c.Config.Client, c.Config.ProjectID, zones, c.Config.LabelSelector)
	if err != nil {
		return success, failed, fmt.Errorf("failed to get instances: %w", err)
	}

	if len(filteredInstances) == 0 {
		success = append(success, common.ScalerStatusSuccess{
			Kind:    "ComputeInstance",
			Name:    "",
			Comment: "No instances found with the label selector",
		})
		return success, failed, nil
	}

	// Process each instance
	for _, instance := range filteredInstances {
		status := common.ScalerStatusSuccess{
			Kind: "ComputeInstance",
			Name: instance.GetName(),
		}

		// Skip instances that are in transitional states
		if gcpUtils.IsInstanceTransitioning(instance) {
			status.Comment = "Instance is in transitional state"
			success = append(success, status)
			continue
		}

		// Determine the desired state based on the period
		desiredState := c.getDesiredState()

		if c.isInstanceInDesiredState(instance, desiredState) {
			status.Comment = "Instance is already in the desired state"
			success = append(success, status)
			continue
		}

		if c.Config.DryRun {
			status.Comment = "Dry run mode"
			success = append(success, status)
			continue
		}

		// Apply the state change
		if err := c.applyInstanceState(ctx, instance, desiredState); err != nil {
			failed = append(failed, common.ScalerStatusFailed{
				Kind:   "ComputeInstance",
				Name:   instance.GetName(),
				Reason: err.Error(),
			})
			continue
		}

		success = append(success, status)
	}

	return success, failed, nil
}

// getDesiredState determines the desired state based on the current period.
// When the period type is "noaction" (e.g. during CR deletion with RestoreOnDelete=true),
// defaultPeriodType is applied — not the pre-CR VM state. This means RestoreOnDelete
// stops VMs by default ("down") unless defaultPeriodType is explicitly set to "up".
func (c *VMInstances) getDesiredState() string {
	return gcpUtils.GetDesiredState(c.Period, c.Config.DefaultPeriodType, gcpUtils.InstanceStopped)
}

// isInstanceInDesiredState checks if the instance is already in the desired state
func (c *VMInstances) isInstanceInDesiredState(instance *computepb.Instance, desiredState string) bool {
	currentState := gcpUtils.GetInstanceStatus(instance)
	return currentState == desiredState
}

// applyInstanceState applies the desired state to the instance
func (c *VMInstances) applyInstanceState(ctx context.Context, instance *computepb.Instance, desiredState string) error {
	zone := c.extractZoneFromInstance(instance)
	if zone == "" {
		return fmt.Errorf("cannot extract zone from instance %s", instance.GetName())
	}

	switch desiredState {
	case gcpUtils.InstanceRunning:
		return c.startInstance(ctx, instance, zone)
	case gcpUtils.InstanceStopped:
		return c.stopInstance(ctx, instance, zone)
	default:
		return fmt.Errorf("unknown desired state: %s", desiredState)
	}
}

// startInstance starts a stopped instance.
func (c *VMInstances) startInstance(ctx context.Context, instance *computepb.Instance, zone string) error {
	if gcpUtils.IsInstanceRunning(instance) {
		return nil
	}
	op, err := c.Config.Client.Instances.Start(ctx, &computepb.StartInstanceRequest{
		Project:  c.Config.ProjectID,
		Zone:     zone,
		Instance: instance.GetName(),
	})
	return c.finalizeInstanceMutation(ctx, op, zone, instance.GetName(), "start", "compute.instances.start", err)
}

// stopInstance stops a running instance.
func (c *VMInstances) stopInstance(ctx context.Context, instance *computepb.Instance, zone string) error {
	if gcpUtils.IsInstanceStopped(instance) {
		return nil
	}
	op, err := c.Config.Client.Instances.Stop(ctx, &computepb.StopInstanceRequest{
		Project:  c.Config.ProjectID,
		Zone:     zone,
		Instance: instance.GetName(),
	})
	return c.finalizeInstanceMutation(ctx, op, zone, instance.GetName(), "stop", "compute.instances.stop", err)
}

func (c *VMInstances) finalizeInstanceMutation(
	ctx context.Context,
	op *compute.Operation,
	zone, instanceName, actionVerb, permissionHint string,
	err error,
) error {
	if err != nil {
		return fmt.Errorf("failed to %s instance %q in zone %q (check %s permission): %w",
			actionVerb, instanceName, zone, permissionHint, err)
	}
	if c.Config.WaitForOperation {
		return gcpUtils.WaitForZoneOperation(ctx, c.Config.Client, c.Config.ProjectID, op.Name(), zone)
	}
	return nil
}

// extractZoneFromInstance extracts the zone from the instance's self link
func (c *VMInstances) extractZoneFromInstance(instance *computepb.Instance) string {
	if instance.GetZone() == "" {
		return ""
	}

	// Extract zone from the zone URL
	// Format: https://www.googleapis.com/compute/v1/projects/PROJECT_ID/zones/ZONE_NAME
	zoneURL := strings.TrimSuffix(instance.GetZone(), "/")

	parts := strings.Split(zoneURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}
