package utils

import (
	"context"
	"fmt"
	"strings"

	compute "google.golang.org/api/compute/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetZonesFromRegion returns a list of zones for a given region
func GetZonesFromRegion(ctx context.Context, client *compute.Service, projectId, region string) ([]string, error) {
	zones, err := client.Zones.List(projectId).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	var regionZones []string
	for _, zone := range zones.Items {
		if strings.HasPrefix(zone.Name, region) {
			regionZones = append(regionZones, zone.Name)
		}
	}

	return regionZones, nil
}

// GetInstancesInZones returns all instances in the specified zones
func GetInstancesInZones(ctx context.Context, client *compute.Service, projectId string, zones []string) ([]*compute.Instance, error) {
	var allInstances []*compute.Instance

	for _, zone := range zones {
		instances, err := client.Instances.List(projectId, zone).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list instances in zone %s: %w", zone, err)
		}

		allInstances = append(allInstances, instances.Items...)
	}

	return allInstances, nil
}

// FilterInstancesByLabels filters instances based on label selector
func FilterInstancesByLabels(instances []*compute.Instance, labelSelector *metaV1.LabelSelector) []*compute.Instance {
	if labelSelector == nil {
		return instances
	}

	var filtered []*compute.Instance
	for _, instance := range instances {
		if matchesLabelSelector(instance, labelSelector) {
			filtered = append(filtered, instance)
		}
	}

	return filtered
}

// matchesLabelSelector checks if an instance matches the label selector
func matchesLabelSelector(instance *compute.Instance, selector *metaV1.LabelSelector) bool {
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
func GetInstanceStatus(instance *compute.Instance) string {
	return instance.Status
}

// IsInstanceRunning checks if an instance is currently running
func IsInstanceRunning(instance *compute.Instance) bool {
	return GetInstanceStatus(instance) == InstanceRunning
}

// IsInstanceStopped checks if an instance is currently stopped
func IsInstanceStopped(instance *compute.Instance) bool {
	return GetInstanceStatus(instance) == InstanceStopped
}

// IsInstanceTransitioning checks if an instance is in a transitional state
func IsInstanceTransitioning(instance *compute.Instance) bool {
	status := GetInstanceStatus(instance)
	return status == InstanceStopping || status == InstanceStarting
}
