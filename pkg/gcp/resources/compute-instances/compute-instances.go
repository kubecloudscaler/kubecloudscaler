package computeinstances

import (
	"context"
	"fmt"
	"strings"
	"time"

	kubecloudscalerv1alpha1 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha1"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	compute "google.golang.org/api/compute/v1"
)

// ComputeInstances handles scaling of GCP Compute Engine instances
type ComputeInstances struct {
	Config *gcpUtils.Config
	Period *period.Period
}

// New creates a new ComputeInstances resource handler
func New(ctx context.Context, config *gcpUtils.Config) (*ComputeInstances, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Client == nil {
		return nil, fmt.Errorf("GCP client cannot be nil")
	}

	if config.ProjectId == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}

	return &ComputeInstances{
		Config: config,
		Period: config.Period,
	}, nil
}

// SetState scales instances based on the current period
func (c *ComputeInstances) SetState(ctx context.Context) ([]kubecloudscalerv1alpha1.ScalerStatusSuccess, []kubecloudscalerv1alpha1.ScalerStatusFailed, error) {
	var (
		success []kubecloudscalerv1alpha1.ScalerStatusSuccess
		failed  []kubecloudscalerv1alpha1.ScalerStatusFailed
	)

	// Get all zones in the region
	zones, err := gcpUtils.GetZonesFromRegion(ctx, c.Config.Client, c.Config.ProjectId, c.Config.Region)
	if err != nil {
		return success, failed, fmt.Errorf("failed to get zones: %w", err)
	}

	// Get all instances in the zones
	instances, err := gcpUtils.GetInstancesInZones(ctx, c.Config.Client, c.Config.ProjectId, zones)
	if err != nil {
		return success, failed, fmt.Errorf("failed to get instances: %w", err)
	}

	// Filter instances by label selector
	filteredInstances := gcpUtils.FilterInstancesByLabels(instances, c.Config.LabelSelector)

	if len(filteredInstances) == 0 {
		success = append(success, kubecloudscalerv1alpha1.ScalerStatusSuccess{
			Kind:    "ComputeInstance",
			Name:    "",
			Comment: "No instances found with the label selector",
		})
		return success, failed, nil
	}

	// Process each instance
	for _, instance := range filteredInstances {
		status := kubecloudscalerv1alpha1.ScalerStatusSuccess{
			Kind: "ComputeInstance",
			Name: instance.Name,
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
			failed = append(failed, kubecloudscalerv1alpha1.ScalerStatusFailed{
				Kind:   "ComputeInstance",
				Name:   instance.Name,
				Reason: err.Error(),
			})
			continue
		}

		success = append(success, status)
	}

	return success, failed, nil
}

// getDesiredState determines the desired state based on the current period
func (c *ComputeInstances) getDesiredState() string {
	if c.Period == nil {
		return gcpUtils.InstanceStopped // Default to stopped if no period
	}

	switch c.Period.Type {
	case "up":
		return gcpUtils.InstanceRunning
	case "down":
		return gcpUtils.InstanceStopped
	default:
		return gcpUtils.InstanceStopped
	}
}

// isInstanceInDesiredState checks if the instance is already in the desired state
func (c *ComputeInstances) isInstanceInDesiredState(instance *compute.Instance, desiredState string) bool {
	currentState := gcpUtils.GetInstanceStatus(instance)
	return currentState == desiredState
}

// applyInstanceState applies the desired state to the instance
func (c *ComputeInstances) applyInstanceState(ctx context.Context, instance *compute.Instance, desiredState string) error {
	zone := c.extractZoneFromInstance(instance)
	if zone == "" {
		return fmt.Errorf("cannot extract zone from instance %s", instance.Name)
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

// startInstance starts a stopped instance
func (c *ComputeInstances) startInstance(ctx context.Context, instance *compute.Instance, zone string) error {
	// Check if instance is already running
	if gcpUtils.IsInstanceRunning(instance) {
		return nil
	}

	// Start the instance
	operation, err := c.Config.Client.Instances.Start(c.Config.ProjectId, zone, instance.Name).Do()
	if err != nil {
		return fmt.Errorf("failed to start instance %s: %w", instance.Name, err)
	}

	// Wait for the operation to complete
	if c.Config.WaitForOperation {
		return c.waitForOperation(ctx, operation, zone)
	}
	return nil
}

// stopInstance stops a running instance
func (c *ComputeInstances) stopInstance(ctx context.Context, instance *compute.Instance, zone string) error {
	// Check if instance is already stopped
	if gcpUtils.IsInstanceStopped(instance) {
		return nil
	}

	// Stop the instance
	operation, err := c.Config.Client.Instances.Stop(c.Config.ProjectId, zone, instance.Name).Do()
	if err != nil {
		return fmt.Errorf("failed to stop instance %s: %w", instance.Name, err)
	}

	// Wait for the operation to complete
	if c.Config.WaitForOperation {
		return c.waitForOperation(ctx, operation, zone)
	}
	return nil
}

// extractZoneFromInstance extracts the zone from the instance's self link
func (c *ComputeInstances) extractZoneFromInstance(instance *compute.Instance) string {
	if instance.Zone == "" {
		return ""
	}

	// Extract zone from the zone URL
	// Format: https://www.googleapis.com/compute/v1/projects/PROJECT_ID/zones/ZONE_NAME
	parts := strings.Split(instance.Zone, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

// waitForOperation waits for a GCP operation to complete
func (c *ComputeInstances) waitForOperation(ctx context.Context, operation *compute.Operation, zone string) error {
	// Set a timeout for the operation
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("operation timed out")
		case <-ticker.C:
			// Check operation status
			op, err := c.Config.Client.ZoneOperations.Get(c.Config.ProjectId, zone, operation.Name).Do()
			if err != nil {
				return fmt.Errorf("failed to get operation status: %w", err)
			}

			if op.Status == "DONE" {
				if op.Error != nil {
					return fmt.Errorf("operation failed: %v", op.Error)
				}
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
