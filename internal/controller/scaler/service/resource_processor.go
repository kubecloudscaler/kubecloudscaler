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

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
	"github.com/rs/zerolog"
)

// ResourceProcessorService handles processing of resources for scaling.
type ResourceProcessorService struct {
	logger *zerolog.Logger
}

// NewResourceProcessorService creates a new ResourceProcessorService.
func NewResourceProcessorService(logger *zerolog.Logger) *ResourceProcessorService {
	return &ResourceProcessorService{
		logger: logger,
	}
}

// ProcessResources processes a list of resources and returns success/failure results.
func (p *ResourceProcessorService) ProcessResources(
	ctx context.Context,
	resourceList []string,
	resourceConfig resources.Config,
) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	var (
		recSuccess []common.ScalerStatusSuccess
		recFailed  []common.ScalerStatusFailed
	)

	for _, resource := range resourceList {
		curResource, err := resources.NewResource(resource, resourceConfig, p.logger)
		if err != nil {
			p.logger.Error().Err(err).Str("resource", resource).Msg("unable to get resource")
			continue
		}

		success, failed, err := curResource.SetState(ctx)
		if err != nil {
			p.logger.Error().Err(err).Str("resource", resource).Msg("unable to set resource state")
			continue
		}

		recSuccess = append(recSuccess, success...)
		recFailed = append(recFailed, failed...)
	}

	return recSuccess, recFailed, nil
}
