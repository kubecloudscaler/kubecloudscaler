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
	"time"

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

// StatusUpdaterService handles updating flow status
type StatusUpdaterService struct {
	client client.Client
	logger *zerolog.Logger
}

// NewStatusUpdaterService creates a new StatusUpdaterService
func NewStatusUpdaterService(client client.Client, logger *zerolog.Logger) *StatusUpdaterService {
	return &StatusUpdaterService{
		client: client,
		logger: logger,
	}
}

// UpdateFlowStatus updates the flow status with the given condition
func (s *StatusUpdaterService) UpdateFlowStatus(
	ctx context.Context,
	flow *kubecloudscalerv1alpha3.Flow,
	condition metav1.Condition,
) (ctrl.Result, error) {
	condition.LastTransitionTime = metav1.NewTime(time.Now())
	flowKey := client.ObjectKeyFromObject(flow)

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := s.client.Get(ctx, flowKey, flow); err != nil {
			return err
		}
		condition.ObservedGeneration = flow.Generation
		s.updateConditionInFlow(flow, condition)
		return s.client.Status().Update(ctx, flow)
	}); err != nil {
		s.logger.Error().Err(err).Msg("unable to update flow status")
		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, err
	}

	s.logger.Info().
		Str("name", flow.Name).
		Str("condition", condition.Type).
		Str("status", string(condition.Status)).
		Msg("flow status updated")

	return ctrl.Result{RequeueAfter: utils.ReconcileSuccessDuration}, nil
}

// updateConditionInFlow updates or adds a condition to the flow
func (s *StatusUpdaterService) updateConditionInFlow(flow *kubecloudscalerv1alpha3.Flow, condition metav1.Condition) {
	conditionIndex := s.findConditionIndex(flow, condition.Type)

	if conditionIndex >= 0 {
		flow.Status.Conditions[conditionIndex] = condition
	} else {
		flow.Status.Conditions = append(flow.Status.Conditions, condition)
	}
}

// findConditionIndex finds the index of a condition by type
func (s *StatusUpdaterService) findConditionIndex(flow *kubecloudscalerv1alpha3.Flow, conditionType string) int {
	for i, c := range flow.Status.Conditions {
		if c.Type == conditionType {
			return i
		}
	}
	return -1
}
