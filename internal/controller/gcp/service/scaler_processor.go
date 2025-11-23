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
	"errors"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	scalerService "github.com/kubecloudscaler/kubecloudscaler/internal/controller/scaler/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	gcpClient "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils/client"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

const (
	// RequeueDelaySeconds is the delay in seconds before requeuing a run-once period.
	RequeueDelaySeconds = 5
)

// ScalerProcessorService handles processing of GCP scaler resources.
type ScalerProcessorService struct {
	periodValidator   scalerService.PeriodValidator
	resourceProcessor *scalerService.ResourceProcessorService
	logger            *zerolog.Logger
}

// NewScalerProcessorService creates a new ScalerProcessorService.
func NewScalerProcessorService(
	periodValidator scalerService.PeriodValidator,
	resourceProcessor *scalerService.ResourceProcessorService,
	logger *zerolog.Logger,
) *ScalerProcessorService {
	return &ScalerProcessorService{
		periodValidator:   periodValidator,
		resourceProcessor: resourceProcessor,
		logger:            logger,
	}
}

// ProcessScaler processes a GCP scaler and returns the results.
func (s *ScalerProcessorService) ProcessScaler(
	ctx context.Context,
	scaler *kubecloudscalerv1alpha3.Gcp,
	secret interface{},
	scalerFinalize bool,
) (ctrl.Result, []common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	// Initialize GCP client for resource operations
	var secretPtr *corev1.Secret
	if secret != nil {
		if s, ok := secret.(*corev1.Secret); ok {
			secretPtr = s
		}
	}
	//nolint:gocritic // gcpClient variable name intentionally shadows imported package for clarity
	gcpClient, err := gcpClient.GetClient(secretPtr, scaler.Spec.Config.ProjectID)
	if err != nil {
		s.logger.Error().Err(err).Msg("unable to get GCP client")
		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil, nil, err
	}

	// Configure resource management settings
	resourceConfig := resources.Config{
		GCP: &gcpUtils.Config{
			Client:            gcpClient,
			ProjectID:         scaler.Spec.Config.ProjectID,
			Region:            scaler.Spec.Config.Region,
			Names:             scaler.Spec.Resources.Names,
			LabelSelector:     scaler.Spec.Resources.LabelSelector,
			DefaultPeriodType: scaler.Spec.Config.DefaultPeriodType,
		},
	}

	// Convert periods for validation
	periods := make([]*common.ScalerPeriod, len(scaler.Spec.Periods))
	for i := range scaler.Spec.Periods {
		periods[i] = &scaler.Spec.Periods[i]
	}

	// Validate and determine the current time period
	resourceConfig.GCP.Period, err = s.periodValidator.ValidatePeriod(
		periods,
		&scaler.Status,
		scaler.Spec.Config.RestoreOnDelete && scalerFinalize,
	)
	if err != nil {
		// Handle run-once period
		if errors.Is(err, utils.ErrRunOncePeriod) {
			return utils.HandleRunOncePeriod(resourceConfig.GCP.Period.GetEndTime, RequeueDelaySeconds), nil, nil, nil
		}
		return ctrl.Result{Requeue: false}, nil, nil, err
	}

	// Check if we should skip noaction period
	if utils.ShouldSkipNoActionPeriod(resourceConfig.GCP.Period.Name, scaler.Status.CurrentPeriod.Name) {
		s.logger.Debug().Msg("no action period, skipping reconciliation")
		return ctrl.Result{RequeueAfter: utils.ReconcileSuccessDuration}, nil, nil, nil
	}

	// Get resource list
	resourceList := s.validResourceList(scaler)
	s.logger.Debug().Msgf("resourceList: %v", resourceList)

	// Process resources
	recSuccess, recFailed, err := s.resourceProcessor.ProcessResources(ctx, resourceList, resourceConfig)
	if err != nil {
		return ctrl.Result{}, recSuccess, recFailed, err
	}

	return ctrl.Result{RequeueAfter: utils.ReconcileSuccessDuration}, recSuccess, recFailed, nil
}

// validResourceList validates and filters the list of resources to be scaled.
func (s *ScalerProcessorService) validResourceList(scaler *kubecloudscalerv1alpha3.Gcp) []string {
	if len(scaler.Spec.Resources.Types) == 0 {
		scaler.Spec.Resources.Types = []string{resources.DefaultGCPResourceType}
	}
	return scaler.Spec.Resources.Types
}
