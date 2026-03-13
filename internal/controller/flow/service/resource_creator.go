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

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/types"
)

// ResourceCreatorService handles creation of K8s and GCP resources
type ResourceCreatorService struct {
	client client.Client
	scheme *runtime.Scheme
	logger *zerolog.Logger
}

// NewResourceCreatorService creates a new ResourceCreatorService
func NewResourceCreatorService(
	client client.Client,
	scheme *runtime.Scheme,
	logger *zerolog.Logger,
) *ResourceCreatorService {
	return &ResourceCreatorService{
		client: client,
		scheme: scheme,
		logger: logger,
	}
}

// CreateK8sResource creates a K8s resource CR with all associated periods
func (c *ResourceCreatorService) CreateK8sResource(
	ctx context.Context,
	flow *kubecloudscalerv1alpha3.Flow,
	resourceName string,
	k8sResource kubecloudscalerv1alpha3.K8sResource,
	periodsWithDelay []types.PeriodWithDelay,
) error {
	allPeriods := c.buildPeriods(periodsWithDelay)

	k8sObj := &kubecloudscalerv1alpha3.K8s{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("flow-%s-%s", flow.Name, resourceName),
			Labels: map[string]string{
				"flow":     flow.Name,
				"resource": resourceName,
			},
		},
		Spec: kubecloudscalerv1alpha3.K8sSpec{
			DryRun:    false,
			Periods:   allPeriods,
			Resources: k8sResource.Resources,
			Config:    k8sResource.Config,
		},
	}

	if err := controllerutil.SetControllerReference(flow, k8sObj, c.scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	if err := c.createOrUpdateResource(ctx, k8sObj); err != nil {
		return fmt.Errorf("failed to create/update K8s object: %w", err)
	}

	c.logger.Info().
		Str("name", k8sObj.Name).
		Int("periods", len(allPeriods)).
		Msg("created/updated K8s resource")

	return nil
}

// CreateGcpResource creates a GCP resource CR with all associated periods
func (c *ResourceCreatorService) CreateGcpResource(
	ctx context.Context,
	flow *kubecloudscalerv1alpha3.Flow,
	resourceName string,
	gcpResource kubecloudscalerv1alpha3.GcpResource,
	periodsWithDelay []types.PeriodWithDelay,
) error {
	allPeriods := c.buildPeriods(periodsWithDelay)

	gcpObj := &kubecloudscalerv1alpha3.Gcp{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("flow-%s-%s", flow.Name, resourceName),
			Labels: map[string]string{
				"flow":     flow.Name,
				"resource": resourceName,
			},
		},
		Spec: kubecloudscalerv1alpha3.GcpSpec{
			DryRun:    false,
			Periods:   allPeriods,
			Resources: gcpResource.Resources,
			Config:    gcpResource.Config,
		},
	}

	if err := controllerutil.SetControllerReference(flow, gcpObj, c.scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	if err := c.createOrUpdateResource(ctx, gcpObj); err != nil {
		return fmt.Errorf("failed to create/update GCP object: %w", err)
	}

	c.logger.Info().
		Str("name", gcpObj.Name).
		Int("periods", len(allPeriods)).
		Msg("created/updated GCP resource")

	return nil
}

// buildPeriods builds periods for K8s resources with adjusted times
func (c *ResourceCreatorService) buildPeriods(periodsWithDelay []types.PeriodWithDelay) []common.ScalerPeriod {
	allPeriods := make([]common.ScalerPeriod, 0, len(periodsWithDelay))
	for _, periodWithDelay := range periodsWithDelay {
		curPeriod := periodWithDelay.Period
		if curPeriod.Time.Recurring != nil {
			copied := *curPeriod.Time.Recurring
			curPeriod.Time.Recurring = &copied
			curPeriod.Time.Recurring.StartTime = periodWithDelay.StartTime.Format("15:04")
			curPeriod.Time.Recurring.EndTime = periodWithDelay.EndTime.Format("15:04")
		}
		if curPeriod.Time.Fixed != nil {
			copied := *curPeriod.Time.Fixed
			curPeriod.Time.Fixed = &copied
			curPeriod.Time.Fixed.StartTime = periodWithDelay.StartTime.Format("2006-01-02 15:04:05")
			curPeriod.Time.Fixed.EndTime = periodWithDelay.EndTime.Format("2006-01-02 15:04:05")
		}
		allPeriods = append(allPeriods, curPeriod)
	}
	return allPeriods
}

// createOrUpdateResource creates or updates a resource using controllerutil.CreateOrUpdate
// to avoid TOCTOU races between Get and Update.
func (c *ResourceCreatorService) createOrUpdateResource(ctx context.Context, obj client.Object) error {
	desired := obj.DeepCopyObject().(client.Object)
	_, err := controllerutil.CreateOrUpdate(ctx, c.client, obj, func() error {
		// After Get, obj may have been overwritten with cluster state.
		// Merge desired labels and annotations with existing ones (preserve existing, overlay desired).
		mergedLabels := obj.GetLabels()
		if mergedLabels == nil {
			mergedLabels = make(map[string]string)
		}
		for k, v := range desired.GetLabels() {
			mergedLabels[k] = v
		}
		obj.SetLabels(mergedLabels)

		mergedAnnotations := obj.GetAnnotations()
		if mergedAnnotations == nil {
			mergedAnnotations = make(map[string]string)
		}
		for k, v := range desired.GetAnnotations() {
			mergedAnnotations[k] = v
		}
		obj.SetAnnotations(mergedAnnotations)
		switch o := obj.(type) {
		case *kubecloudscalerv1alpha3.K8s:
			o.Spec = desired.(*kubecloudscalerv1alpha3.K8s).Spec
		case *kubecloudscalerv1alpha3.Gcp:
			o.Spec = desired.(*kubecloudscalerv1alpha3.Gcp).Spec
		default:
			return fmt.Errorf("unsupported object type %T", obj)
		}
		return nil
	})
	return err
}
