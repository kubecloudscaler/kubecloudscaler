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
	"slices"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	scalerService "github.com/kubecloudscaler/kubecloudscaler/internal/controller/scaler/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	k8sUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	k8sClient "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils/client"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

const (
	// RequeueDelaySeconds is the delay in seconds before requeuing a run-once period.
	RequeueDelaySeconds = 5
)

// ScalerProcessorService handles processing of K8s scaler resources.
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

// ProcessScaler processes a K8s scaler and returns the results.
func (s *ScalerProcessorService) ProcessScaler(
	ctx context.Context,
	scaler *kubecloudscalerv1alpha3.K8s,
	secret interface{},
	scalerFinalize bool,
) (ctrl.Result, []common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	// Initialize Kubernetes client for resource operations
	var secretPtr *corev1.Secret
	if secret != nil {
		if s, ok := secret.(*corev1.Secret); ok {
			secretPtr = s
		}
	}
	kubeClient, dynamicClient, err := k8sClient.GetClient(secretPtr)
	if err != nil {
		s.logger.Error().Err(err).Msg("unable to get k8s client")
		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil, nil, err
	}

	// Configure resource management settings
	resourceConfig := resources.Config{
		K8s: &k8sUtils.Config{
			Client:                       kubeClient,
			DynamicClient:                dynamicClient,
			Names:                        scaler.Spec.Resources.Names,
			Namespaces:                   scaler.Spec.Config.Namespaces,
			ExcludeNamespaces:            scaler.Spec.Config.ExcludeNamespaces,
			LabelSelector:                scaler.Spec.Resources.LabelSelector,
			ForceExcludeSystemNamespaces: scaler.Spec.Config.ForceExcludeSystemNamespaces,
		},
	}

	// Convert periods for validation
	periods := make([]*common.ScalerPeriod, len(scaler.Spec.Periods))
	for i := range scaler.Spec.Periods {
		periods[i] = &scaler.Spec.Periods[i]
	}

	// Validate and determine the current time period
	resourceConfig.K8s.Period, err = s.periodValidator.ValidatePeriod(
		periods,
		&scaler.Status,
		scaler.Spec.Config.RestoreOnDelete && scalerFinalize,
	)
	if err != nil {
		// Handle run-once period
		if errors.Is(err, utils.ErrRunOncePeriod) {
			return utils.HandleRunOncePeriod(resourceConfig.K8s.Period.GetEndTime, RequeueDelaySeconds), nil, nil, nil
		}
		return ctrl.Result{Requeue: false}, nil, nil, err
	}

	// Check if we should skip noaction period
	if utils.ShouldSkipNoActionPeriod(resourceConfig.K8s.Period.Name, scaler.Status.CurrentPeriod.Name) {
		s.logger.Debug().Msg("no action period, skipping reconciliation")
		return ctrl.Result{RequeueAfter: utils.ReconcileSuccessDuration}, nil, nil, nil
	}

	// Validate resource list
	resourceList, err := s.validResourceList(scaler)
	if err != nil {
		return ctrl.Result{}, nil, nil, err
	}

	// Process resources
	recSuccess, recFailed, err := s.resourceProcessor.ProcessResources(ctx, resourceList, resourceConfig)
	if err != nil {
		return ctrl.Result{}, recSuccess, recFailed, err
	}

	return ctrl.Result{RequeueAfter: utils.ReconcileSuccessDuration}, recSuccess, recFailed, nil
}

// validResourceList validates and filters the list of resources to be scaled.
func (s *ScalerProcessorService) validResourceList(scaler *kubecloudscalerv1alpha3.K8s) ([]string, error) {
	output := make([]string, 0, len(scaler.Spec.Resources.Types))
	var (
		isApp bool
		isHpa bool
	)

	if len(scaler.Spec.Resources.Types) == 0 {
		scaler.Spec.Resources.Types = []string{resources.DefaultK8SResourceType}
	}

	for _, resource := range scaler.Spec.Resources.Types {
		if slices.Contains(utils.AppsResources, resource) {
			isApp = true
		}
		if slices.Contains(utils.HpaResources, resource) {
			isHpa = true
		}
		if isHpa && isApp {
			s.logger.Info().Msg(utils.ErrMixedAppsHPA.Error())
			return []string{}, utils.ErrMixedAppsHPA
		}
		output = append(output, resource)
	}

	return output, nil
}
