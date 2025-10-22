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
	"context"
	"fmt"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/types"
	"github.com/rs/zerolog"
)

// FlowProcessorService handles the business logic for processing flows
type FlowProcessorService struct {
	validator       FlowValidator
	resourceMapper  ResourceMapper
	resourceCreator ResourceCreator
	logger          *zerolog.Logger
}

// NewFlowProcessorService creates a new FlowProcessorService
func NewFlowProcessorService(
	validator FlowValidator,
	resourceMapper ResourceMapper,
	resourceCreator ResourceCreator,
	logger *zerolog.Logger,
) *FlowProcessorService {
	return &FlowProcessorService{
		validator:       validator,
		resourceMapper:  resourceMapper,
		resourceCreator: resourceCreator,
		logger:          logger,
	}
}

// ProcessFlow processes the flow definition and creates/deploys resources
func (s *FlowProcessorService) ProcessFlow(ctx context.Context, flow *kubecloudscalerv1alpha3.Flow) error {
	s.logger.Info().
		Str("flow", flow.Name).
		Msg("processing flow")

	// Extract flow data
	resourceNames, periodNames, err := s.validator.ExtractFlowData(flow)
	if err != nil {
		return fmt.Errorf("failed to extract flow data: %w", err)
	}

	// Validate timing constraints
	if err := s.validator.ValidatePeriodTimings(flow, periodNames); err != nil {
		return fmt.Errorf("period timing validation failed: %w", err)
	}

	// Create resource mappings
	resourceMappings, err := s.resourceMapper.CreateResourceMappings(flow, resourceNames)
	if err != nil {
		return fmt.Errorf("failed to create resource mappings: %w", err)
	}

	// Process each resource
	for resourceName, resourceInfo := range resourceMappings {
		if err := s.processResource(ctx, flow, resourceName, resourceInfo); err != nil {
			return fmt.Errorf("failed to process resource %s: %w", resourceName, err)
		}
	}

	s.logger.Info().
		Str("flow", flow.Name).
		Int("resources", len(resourceMappings)).
		Msg("flow processed successfully")

	return nil
}

// processResource processes a single resource
func (s *FlowProcessorService) processResource(
	ctx context.Context,
	flow *kubecloudscalerv1alpha3.Flow,
	resourceName string,
	resourceInfo types.ResourceInfo,
) error {
	switch resourceInfo.Type {
	case "k8s":
		k8sResource, ok := resourceInfo.Resource.(kubecloudscalerv1alpha3.K8sResource)
		if !ok {
			return fmt.Errorf("expected K8sResource, got %T", resourceInfo.Resource)
		}
		return s.resourceCreator.CreateK8sResource(ctx, flow, resourceName, k8sResource, resourceInfo.Periods)
	case "gcp":
		gcpResource, ok := resourceInfo.Resource.(kubecloudscalerv1alpha3.GcpResource)
		if !ok {
			return fmt.Errorf("expected GcpResource, got %T", resourceInfo.Resource)
		}
		return s.resourceCreator.CreateGcpResource(ctx, flow, resourceName, gcpResource, resourceInfo.Periods)
	default:
		return fmt.Errorf("unknown resource type: %s", resourceInfo.Type)
	}
}
