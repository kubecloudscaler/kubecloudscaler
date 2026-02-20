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

package service

import (
	"fmt"
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/types"
	"github.com/rs/zerolog"
)

// ResourceMapperService handles mapping of resources with their associated periods
type ResourceMapperService struct {
	timeCalculator TimeCalculator
	logger         *zerolog.Logger
}

// NewResourceMapperService creates a new ResourceMapperService
func NewResourceMapperService(timeCalculator TimeCalculator, logger *zerolog.Logger) *ResourceMapperService {
	return &ResourceMapperService{
		timeCalculator: timeCalculator,
		logger:         logger,
	}
}

// CreateResourceMappings creates mappings for all resources with their associated periods
func (m *ResourceMapperService) CreateResourceMappings(
	flow *kubecloudscalerv1alpha3.Flow,
	resourceNames map[string]bool,
) (map[string]types.ResourceInfo, error) {
	resourceMappings := make(map[string]types.ResourceInfo)
	periodsMap := m.createPeriodsMap(flow)

	for resourceName := range resourceNames {
		resourceInfo, err := m.mapResource(flow, resourceName, periodsMap)
		if err != nil {
			return nil, fmt.Errorf("failed to map resource %s: %w", resourceName, err)
		}
		resourceMappings[resourceName] = resourceInfo
	}

	return resourceMappings, nil
}

// createPeriodsMap creates a map of period names to periods
func (m *ResourceMapperService) createPeriodsMap(flow *kubecloudscalerv1alpha3.Flow) map[string]common.ScalerPeriod {
	periodsMap := make(map[string]common.ScalerPeriod)
	for i := range flow.Spec.Periods {
		period := flow.Spec.Periods[i]
		periodsMap[period.Name] = period
	}
	return periodsMap
}

// mapResource maps a single resource with its associated periods
func (m *ResourceMapperService) mapResource(
	flow *kubecloudscalerv1alpha3.Flow,
	resourceName string,
	periodsMap map[string]common.ScalerPeriod,
) (types.ResourceInfo, error) {
	// Find the resource in K8s resources
	k8sResource := m.findK8sResource(flow, resourceName)
	gcpResource := m.findGcpResource(flow, resourceName)

	// Determine resource type and validate uniqueness
	resourceType, resourceObj, err := m.determineResourceType(resourceName, k8sResource, gcpResource)
	if err != nil {
		return types.ResourceInfo{}, err
	}

	// Find all periods associated with this resource
	periodsWithDelay, err := m.findAssociatedPeriods(flow, resourceName, periodsMap)
	if err != nil {
		return types.ResourceInfo{}, fmt.Errorf("failed to find associated periods: %w", err)
	}

	return types.ResourceInfo{
		Type:     resourceType,
		Resource: resourceObj,
		Periods:  periodsWithDelay,
	}, nil
}

// findK8sResource finds a K8s resource by name
func (m *ResourceMapperService) findK8sResource(flow *kubecloudscalerv1alpha3.Flow, resourceName string) *kubecloudscalerv1alpha3.K8sResource {
	for _, resource := range flow.Spec.Resources.K8s {
		if resource.Name == resourceName {
			return &resource
		}
	}
	return nil
}

// findGcpResource finds a GCP resource by name
func (m *ResourceMapperService) findGcpResource(flow *kubecloudscalerv1alpha3.Flow, resourceName string) *kubecloudscalerv1alpha3.GcpResource {
	for _, resource := range flow.Spec.Resources.Gcp {
		if resource.Name == resourceName {
			return &resource
		}
	}
	return nil
}

// determineResourceType determines the resource type and validates uniqueness
func (m *ResourceMapperService) determineResourceType(
	resourceName string,
	k8sResource *kubecloudscalerv1alpha3.K8sResource,
	gcpResource *kubecloudscalerv1alpha3.GcpResource,
) (string, interface{}, error) {
	if k8sResource != nil && gcpResource != nil {
		return "", nil, fmt.Errorf("resource %s is defined in both K8s and GCP resources", resourceName)
	}

	if k8sResource != nil {
		return "k8s", *k8sResource, nil
	}

	if gcpResource != nil {
		return "gcp", *gcpResource, nil
	}

	return "", nil, fmt.Errorf("resource %s referenced in flows but not defined in resources", resourceName)
}

// findAssociatedPeriods finds all periods associated with a resource
func (m *ResourceMapperService) findAssociatedPeriods(
	flow *kubecloudscalerv1alpha3.Flow,
	resourceName string,
	periodsMap map[string]common.ScalerPeriod,
) ([]types.PeriodWithDelay, error) {
	var periodsWithDelay []types.PeriodWithDelay
	seen := make(map[string]bool)

	for _, flowItem := range flow.Spec.Flows {
		for _, resource := range flowItem.Resources {
			if resource.Name != resourceName {
				continue
			}

			key := flowItem.PeriodName + "/" + resource.Name
			if seen[key] {
				return nil, fmt.Errorf(
					"resource %s appears more than once for period %s in flows",
					resource.Name, flowItem.PeriodName,
				)
			}
			seen[key] = true

			period, exists := periodsMap[flowItem.PeriodName]
			if !exists {
				return nil, fmt.Errorf("period %s referenced in flows but not defined", flowItem.PeriodName)
			}

			periodWithDelay, err := m.createPeriodWithDelay(&period, &resource)
			if err != nil {
				return nil, fmt.Errorf("failed to create period with delay: %w", err)
			}

			periodsWithDelay = append(periodsWithDelay, periodWithDelay)
		}
	}

	return periodsWithDelay, nil
}

// createPeriodWithDelay creates a PeriodWithDelay from a period and resource
func (m *ResourceMapperService) createPeriodWithDelay(
	period *common.ScalerPeriod,
	resource *kubecloudscalerv1alpha3.FlowResource,
) (types.PeriodWithDelay, error) {
	startTimeDelay, err := m.parseDelay(resource.StartTimeDelay)
	if err != nil {
		return types.PeriodWithDelay{}, fmt.Errorf("invalid start time delay format for resource %s: %w", resource.Name, err)
	}

	endTimeDelay, err := m.parseDelay(resource.EndTimeDelay)
	if err != nil {
		return types.PeriodWithDelay{}, fmt.Errorf("invalid end time delay format for resource %s: %w", resource.Name, err)
	}

	startTime, err := m.timeCalculator.CalculatePeriodStartTime(period, startTimeDelay)
	if err != nil {
		return types.PeriodWithDelay{}, fmt.Errorf("failed to calculate start time for period: %w", err)
	}

	endTime, err := m.timeCalculator.CalculatePeriodEndTime(period, endTimeDelay)
	if err != nil {
		return types.PeriodWithDelay{}, fmt.Errorf("failed to calculate end time for period: %w", err)
	}

	return types.PeriodWithDelay{
		Period:         *period,
		StartTimeDelay: startTimeDelay,
		EndTimeDelay:   endTimeDelay,
		StartTime:      startTime,
		EndTime:        endTime,
	}, nil
}

// parseDelay parses a delay string into a duration
func (m *ResourceMapperService) parseDelay(delayStr string) (time.Duration, error) {
	if delayStr == "" {
		return 0, nil
	}
	return time.ParseDuration(delayStr)
}
