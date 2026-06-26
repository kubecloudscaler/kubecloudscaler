// Package instancegroupmanagers provides MIG scaling functionality for GCP resources.
package instancegroupmanagers

import (
	"context"
	"fmt"
	"slices"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
)

const (
	// managedInstanceActionNone is the CurrentAction value when the MIG has no pending work on an instance.
	managedInstanceActionNone = "NONE"

	// migInstanceStopped is the ManagedInstance.InstanceStatus value after instanceGroupManagers.stopInstances.
	// This is distinct from gcpUtils.InstanceStopped ("TERMINATED") which is the Instance.Status value
	// produced by plain instances.stop. The ManagedInstance proto uses a separate enum where
	// stopInstances yields "STOPPED", not "TERMINATED".
	migInstanceStopped = "STOPPED"
)

// SetState stops or starts all managed instances in the selected MIGs based on the current period.
// MIGs are selected by name (Config.Names); if empty, all MIGs in the region are processed.
// LabelSelector is not used: MIG resources do not expose labels at the resource level.
func (c *InstanceGroupManagers) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	success := make([]common.ScalerStatusSuccess, 0)
	failed := make([]common.ScalerStatusFailed, 0)

	zones, err := gcpUtils.GetZonesFromRegion(ctx, c.Config.Client, c.Config.ProjectID, c.Config.Region)
	if err != nil {
		return success, failed, fmt.Errorf("failed to get zones: %w", err)
	}

	desiredState := c.getDesiredState()
	found := false

	for _, zone := range zones {
		migs, err := c.listMIGsInZone(ctx, zone)
		if err != nil {
			return success, failed, fmt.Errorf("failed to list instance group managers in zone %q: %w", zone, err)
		}

		for _, mig := range migs {
			found = true
			migSuccess, migFailed := c.processMIG(ctx, mig, zone, desiredState)
			success = append(success, migSuccess...)
			failed = append(failed, migFailed...)
		}
	}

	if !found {
		success = append(success, common.ScalerStatusSuccess{
			Kind:    "InstanceGroupManager",
			Name:    "",
			Comment: "No instance group managers found",
		})
	}

	return success, failed, nil
}

// listMIGsInZone lists all MIGs in a zone and filters by Config.Names when set.
func (c *InstanceGroupManagers) listMIGsInZone(ctx context.Context, zone string) ([]*computepb.InstanceGroupManager, error) {
	it := c.Config.Client.InstanceGroupManagers.List(ctx, &computepb.ListInstanceGroupManagersRequest{
		Project: c.Config.ProjectID,
		Zone:    zone,
	})

	var migs []*computepb.InstanceGroupManager
	for {
		mig, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(
				"failed to list instance group managers in zone %q of project %q "+
					"(check that the service account has compute.instanceGroupManagers.list permission): %w",
				zone, c.Config.ProjectID, err,
			)
		}
		if c.isMIGSelected(mig) {
			migs = append(migs, mig)
		}
	}

	return migs, nil
}

// isMIGSelected returns true if the MIG should be managed. When no names are configured, all MIGs are selected.
func (c *InstanceGroupManagers) isMIGSelected(mig *computepb.InstanceGroupManager) bool {
	if len(c.Config.Names) == 0 {
		return true
	}
	return slices.Contains(c.Config.Names, mig.GetName())
}

// processMIG lists managed instances in a MIG, determines which ones need a state change,
// then calls StopInstances or StartInstances on the MIG with those instance URLs.
func (c *InstanceGroupManagers) processMIG(
	ctx context.Context,
	mig *computepb.InstanceGroupManager,
	zone, desiredState string,
) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed) {
	status := common.ScalerStatusSuccess{
		Kind: "InstanceGroupManager",
		Name: mig.GetName(),
	}

	instances, err := c.listManagedInstances(ctx, mig.GetName(), zone)
	if err != nil {
		return nil, []common.ScalerStatusFailed{{
			Kind:   "InstanceGroupManager",
			Name:   mig.GetName(),
			Reason: err.Error(),
		}}
	}

	// Collect instance URLs that need a state change and are not currently transitioning.
	toChange := make([]string, 0, len(instances))
	for _, inst := range instances {
		if isManagedInstanceTransitioning(inst) {
			continue
		}
		if !isManagedInstanceInDesiredState(inst, desiredState) {
			toChange = append(toChange, inst.GetInstance())
		}
	}

	if len(toChange) == 0 {
		status.Comment = "All instances are already in the desired state"
		return []common.ScalerStatusSuccess{status}, nil
	}

	if c.Config.DryRun {
		status.Comment = "Dry run mode"
		return []common.ScalerStatusSuccess{status}, nil
	}

	if err := c.applyMIGState(ctx, mig.GetName(), zone, desiredState, toChange); err != nil {
		return nil, []common.ScalerStatusFailed{{
			Kind:   "InstanceGroupManager",
			Name:   mig.GetName(),
			Reason: err.Error(),
		}}
	}

	return []common.ScalerStatusSuccess{status}, nil
}

// listManagedInstances returns all managed instances in a MIG.
func (c *InstanceGroupManagers) listManagedInstances(
	ctx context.Context,
	migName, zone string,
) ([]*computepb.ManagedInstance, error) {
	it := c.Config.Client.InstanceGroupManagers.ListManagedInstances(
		ctx,
		&computepb.ListManagedInstancesInstanceGroupManagersRequest{
			InstanceGroupManager: migName,
			Project:              c.Config.ProjectID,
			Zone:                 zone,
		},
	)

	var instances []*computepb.ManagedInstance
	for {
		inst, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf(
				"failed to list managed instances for MIG %q in zone %q "+
					"(check compute.instanceGroupManagers.list permission): %w",
				migName, zone, err,
			)
		}
		instances = append(instances, inst)
	}

	return instances, nil
}

// applyMIGState calls StopInstances or StartInstances on the MIG for the given instance URLs.
func (c *InstanceGroupManagers) applyMIGState(
	ctx context.Context,
	migName, zone, desiredState string,
	instances []string,
) error {
	switch desiredState {
	case gcpUtils.InstanceRunning:
		return c.startMIGInstances(ctx, migName, zone, instances)
	case migInstanceStopped:
		return c.stopMIGInstances(ctx, migName, zone, instances)
	default:
		return fmt.Errorf("unknown desired state: %s", desiredState)
	}
}

// stopMIGInstances calls instanceGroupManagers.stopInstances. The MIG autohealer stands down
// and boot disks are preserved (unlike a plain instances.stop which triggers RECREATING).
func (c *InstanceGroupManagers) stopMIGInstances(
	ctx context.Context,
	migName, zone string,
	instances []string,
) error {
	op, err := c.Config.Client.InstanceGroupManagers.StopInstances(
		ctx,
		&computepb.StopInstancesInstanceGroupManagerRequest{
			InstanceGroupManager: migName,
			Project:              c.Config.ProjectID,
			Zone:                 zone,
			InstanceGroupManagersStopInstancesRequestResource: &computepb.InstanceGroupManagersStopInstancesRequest{
				Instances: instances,
			},
		},
	)
	return c.finalizeIGMMutation(ctx, op, zone, migName, "stop", "compute.instanceGroupManagers.stopInstances", err)
}

// startMIGInstances calls instanceGroupManagers.startInstances.
func (c *InstanceGroupManagers) startMIGInstances(
	ctx context.Context,
	migName, zone string,
	instances []string,
) error {
	op, err := c.Config.Client.InstanceGroupManagers.StartInstances(
		ctx,
		&computepb.StartInstancesInstanceGroupManagerRequest{
			InstanceGroupManager: migName,
			Project:              c.Config.ProjectID,
			Zone:                 zone,
			InstanceGroupManagersStartInstancesRequestResource: &computepb.InstanceGroupManagersStartInstancesRequest{
				Instances: instances,
			},
		},
	)
	return c.finalizeIGMMutation(ctx, op, zone, migName, "start", "compute.instanceGroupManagers.startInstances", err)
}

func (c *InstanceGroupManagers) finalizeIGMMutation(
	ctx context.Context,
	op *compute.Operation,
	zone, migName, actionVerb, permissionHint string,
	err error,
) error {
	if err != nil {
		return fmt.Errorf("failed to %s instances in MIG %q in zone %q (check %s permission): %w",
			actionVerb, migName, zone, permissionHint, err)
	}
	if c.Config.WaitForOperation {
		return gcpUtils.WaitForZoneOperation(ctx, c.Config.Client, c.Config.ProjectID, op.Name(), zone)
	}
	return nil
}

// getDesiredState determines the desired state based on the current period.
// Uses migInstanceStopped ("STOPPED") — not gcpUtils.InstanceStopped ("TERMINATED") — because
// ManagedInstance.InstanceStatus uses a distinct proto enum from Instance.Status.
func (c *InstanceGroupManagers) getDesiredState() string {
	return gcpUtils.GetDesiredState(c.Period, c.Config.DefaultPeriodType, migInstanceStopped)
}

// isManagedInstanceTransitioning returns true when the MIG has a pending action on the instance.
// Instances in transition should be skipped to avoid conflicting operations.
func isManagedInstanceTransitioning(inst *computepb.ManagedInstance) bool {
	return inst.GetCurrentAction() != managedInstanceActionNone
}

// isManagedInstanceInDesiredState returns true when the instance already matches desiredState.
func isManagedInstanceInDesiredState(inst *computepb.ManagedInstance, desiredState string) bool {
	return inst.GetInstanceStatus() == desiredState
}
