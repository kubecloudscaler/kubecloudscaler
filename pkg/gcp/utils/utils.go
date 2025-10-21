// Package utils provides utility functions for GCP resource management in the kubecloudscaler project.
package utils

import (
	"context"
	"fmt"
	"strings"

	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetZonesFromRegion returns a list of zones for a given region using Regions API
func GetZonesFromRegion(ctx context.Context, clients *ClientSet, projectID, region string) ([]string, error) {
	if clients == nil || clients.Regions == nil {
		return nil, fmt.Errorf("regions client is nil")
	}

	req := &computepb.GetRegionRequest{
		Project: projectID,
		Region:  region,
	}

	reg, err := clients.Regions.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get region %s: %w", region, err)
	}

	var zones []string
	for _, z := range reg.Zones {
		// z is a full URL; extract the last segment as the zone name
		parts := strings.Split(z, "/")
		if len(parts) > 0 {
			zones = append(zones, parts[len(parts)-1])
		}
	}

	return zones, nil
}

// GetInstancesInZones returns all instances in the specified zones using apiv1
// If labelSelector is provided, it will be converted to a GCP filter and applied at the API level
func GetInstancesInZones(
	ctx context.Context,
	clients *ClientSet,
	projectID string,
	zones []string,
	labelSelector *metaV1.LabelSelector,
) ([]*computepb.Instance, error) {
	if clients == nil || clients.Instances == nil {
		return nil, fmt.Errorf("instances client is nil")
	}

	// Build filter from label selector
	filter := buildGCPFilterFromLabelSelector(labelSelector)

	var allInstances []*computepb.Instance
	for _, zone := range zones {
		req := &computepb.ListInstancesRequest{
			Project: projectID,
			Zone:    zone,
		}

		// Add filter if we have one
		if filter != "" {
			req.Filter = &filter
		}

		it := clients.Instances.List(ctx, req)
		for {
			inst, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to list instances in zone %s: %w", zone, err)
			}
			allInstances = append(allInstances, inst)
		}
	}

	return allInstances, nil
}

// buildGCPFilterFromLabelSelector converts a Kubernetes label selector to a GCP filter string
// GCP filter format: labels.key=value AND labels.key2=value2
func buildGCPFilterFromLabelSelector(labelSelector *metaV1.LabelSelector) string {
	if labelSelector == nil || len(labelSelector.MatchLabels) == 0 {
		return ""
	}

	filters := make([]string, 0, len(labelSelector.MatchLabels))
	for key, value := range labelSelector.MatchLabels {
		// GCP filter format for labels: labels.key=value
		filters = append(filters, fmt.Sprintf("labels.%s=%s", key, value))
	}

	// Join multiple filters with AND
	return strings.Join(filters, " AND ")
}

// FilterInstancesByLabels filters instances based on label selector
// This is now primarily used for additional filtering or when a filter wasn't applied at the API level
func FilterInstancesByLabels(instances []*computepb.Instance, labelSelector *metaV1.LabelSelector) []*computepb.Instance {
	if labelSelector == nil {
		return instances
	}

	var filtered []*computepb.Instance
	for _, instance := range instances {
		if matchesLabelSelector(instance, labelSelector) {
			filtered = append(filtered, instance)
		}
	}

	return filtered
}

// matchesLabelSelector checks if an instance matches the label selector
func matchesLabelSelector(instance *computepb.Instance, selector *metaV1.LabelSelector) bool {
	if instance.Labels == nil {
		return false
	}

	// Check matchLabels
	for key, value := range selector.MatchLabels {
		if instance.Labels[key] != value {
			return false
		}
	}

	// TODO: Implement matchExpressions for more complex label matching
	// For now, we only support matchLabels

	return true
}

// GetInstanceStatus returns the current status of an instance
func GetInstanceStatus(instance *computepb.Instance) string {
	if instance.Status == nil {
		return ""
	}
	return *instance.Status
}

// IsInstanceRunning checks if an instance is currently running
func IsInstanceRunning(instance *computepb.Instance) bool {
	return GetInstanceStatus(instance) == InstanceRunning
}

// IsInstanceStopped checks if an instance is currently stopped
func IsInstanceStopped(instance *computepb.Instance) bool {
	return GetInstanceStatus(instance) == InstanceStopped
}

// IsInstanceTransitioning checks if an instance is in a transitional state
func IsInstanceTransitioning(instance *computepb.Instance) bool {
	status := GetInstanceStatus(instance)
	return status == InstanceStopping || status == InstanceStarting
}
