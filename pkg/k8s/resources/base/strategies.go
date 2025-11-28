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

package base

import (
	"context"

	"github.com/rs/zerolog"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

const (
	periodTypeDown = "down"
)

// IntReplicasStrategy handles scaling for resources with integer replicas (Deployments, StatefulSets).
type IntReplicasStrategy struct {
	kind          string
	getReplicas   func(ResourceItem) *int32
	setReplicas   func(ResourceItem, *int32)
	logger        *zerolog.Logger
	annotationMgr utils.AnnotationManager
}

// NewIntReplicasStrategy creates a new IntReplicasStrategy.
func NewIntReplicasStrategy(
	kind string,
	getReplicas func(ResourceItem) *int32,
	setReplicas func(ResourceItem, *int32),
	logger *zerolog.Logger,
	annotationMgr utils.AnnotationManager,
) *IntReplicasStrategy {
	return &IntReplicasStrategy{
		kind:          kind,
		getReplicas:   getReplicas,
		setReplicas:   setReplicas,
		logger:        logger,
		annotationMgr: annotationMgr,
	}
}

// GetKind returns the resource kind.
func (s *IntReplicasStrategy) GetKind() string {
	return s.kind
}

// ApplyScaling applies scaling logic for integer replicas.
func (s *IntReplicasStrategy) ApplyScaling(ctx context.Context, resource ResourceItem, periodType string, period *periodPkg.Period) (bool, error) {
	switch periodType {
	case periodTypeDown:
		s.logger.Debug().Msgf("scaling down: %s", resource.GetName())
		currentReplicas := s.getReplicas(resource)
		resource.SetAnnotations(s.annotationMgr.AddIntAnnotations(resource.GetAnnotations(), period, currentReplicas))
		s.setReplicas(resource, ptr.To(period.MinReplicas))

	case "up":
		s.logger.Debug().Msgf("scaling up: %s", resource.GetName())
		currentReplicas := s.getReplicas(resource)
		resource.SetAnnotations(s.annotationMgr.AddIntAnnotations(resource.GetAnnotations(), period, currentReplicas))
		s.setReplicas(resource, ptr.To(period.MaxReplicas))

	default:
		s.logger.Debug().Msgf("restoring: %s", resource.GetName())
		isAlreadyRestored, replicas, annotations, err := s.annotationMgr.RestoreIntAnnotations(resource.GetAnnotations())
		if err != nil {
			return false, err
		}

		if isAlreadyRestored {
			return true, nil
		}

		s.setReplicas(resource, replicas)
		resource.SetAnnotations(annotations)
	}

	return false, nil
}

// MinMaxReplicasStrategy handles scaling for resources with min/max replicas (HPA, ARS).
type MinMaxReplicasStrategy struct {
	kind              string
	getMinMaxReplicas func(ResourceItem) (*int32, *int32)
	setMinMaxReplicas func(ResourceItem, *int32, *int32)
	logger            *zerolog.Logger
	annotationMgr     utils.AnnotationManager
}

// NewMinMaxReplicasStrategy creates a new MinMaxReplicasStrategy.
func NewMinMaxReplicasStrategy(
	kind string,
	getMinMaxReplicas func(ResourceItem) (*int32, *int32),
	setMinMaxReplicas func(ResourceItem, *int32, *int32),
	logger *zerolog.Logger,
	annotationMgr utils.AnnotationManager,
) *MinMaxReplicasStrategy {
	return &MinMaxReplicasStrategy{
		kind:              kind,
		getMinMaxReplicas: getMinMaxReplicas,
		setMinMaxReplicas: setMinMaxReplicas,
		logger:            logger,
		annotationMgr:     annotationMgr,
	}
}

// GetKind returns the resource kind.
func (s *MinMaxReplicasStrategy) GetKind() string {
	return s.kind
}

// ApplyScaling applies scaling logic for min/max replicas.
func (s *MinMaxReplicasStrategy) ApplyScaling(ctx context.Context, resource ResourceItem, periodType string, period *periodPkg.Period) (bool, error) {
	switch periodType {
	case periodTypeDown, "up":
		s.logger.Debug().Msgf("scaling %s: %s", periodType, resource.GetName())
		minReplicas, maxReplicas := s.getMinMaxReplicas(resource)
		resource.SetAnnotations(s.annotationMgr.AddMinMaxAnnotations(
			resource.GetAnnotations(),
			period,
			minReplicas,
			ptr.Deref(maxReplicas, 0),
		))
		s.setMinMaxReplicas(resource, ptr.To(period.MinReplicas), ptr.To(period.MaxReplicas))

	default:
		s.logger.Debug().Msgf("restoring: %s", resource.GetName())
		isAlreadyRestored, minReplicas, maxReplicas, annotations, err := s.annotationMgr.RestoreMinMaxAnnotations(resource.GetAnnotations())
		if err != nil {
			return false, err
		}

		if isAlreadyRestored {
			return true, nil
		}

		s.setMinMaxReplicas(resource, minReplicas, ptr.To(maxReplicas))
		resource.SetAnnotations(annotations)
	}

	return false, nil
}

// BoolSuspendStrategy handles scaling for resources with boolean suspend (CronJobs).
type BoolSuspendStrategy struct {
	kind          string
	getSuspend    func(ResourceItem) *bool
	setSuspend    func(ResourceItem, *bool)
	suspended     bool
	logger        *zerolog.Logger
	onUpError     func(ResourceItem) error // Called when trying to scale up (not supported)
	annotationMgr utils.AnnotationManager
}

// NewBoolSuspendStrategy creates a new BoolSuspendStrategy.
func NewBoolSuspendStrategy(
	kind string,
	getSuspend func(ResourceItem) *bool,
	setSuspend func(ResourceItem, *bool),
	suspended bool,
	logger *zerolog.Logger,
	onUpError func(ResourceItem) error,
	annotationMgr utils.AnnotationManager,
) *BoolSuspendStrategy {
	return &BoolSuspendStrategy{
		kind:          kind,
		getSuspend:    getSuspend,
		setSuspend:    setSuspend,
		suspended:     suspended,
		logger:        logger,
		onUpError:     onUpError,
		annotationMgr: annotationMgr,
	}
}

// GetKind returns the resource kind.
func (s *BoolSuspendStrategy) GetKind() string {
	return s.kind
}

// ApplyScaling applies scaling logic for boolean suspend.
func (s *BoolSuspendStrategy) ApplyScaling(ctx context.Context, resource ResourceItem, periodType string, period *periodPkg.Period) (bool, error) {
	switch periodType {
	case periodTypeDown:
		s.logger.Debug().Msgf("scaling down: %s", resource.GetName())
		currentSuspend := s.getSuspend(resource)
		resource.SetAnnotations(s.annotationMgr.AddBoolAnnotations(
			resource.GetAnnotations(),
			period,
			ptr.Deref(currentSuspend, false),
		))
		s.setSuspend(resource, ptr.To(s.suspended))

	case "up":
		if s.onUpError != nil {
			return false, s.onUpError(resource)
		}
		s.logger.Debug().Msgf("scaling up: %s", resource.GetName())
		currentSuspend := s.getSuspend(resource)
		resource.SetAnnotations(s.annotationMgr.AddBoolAnnotations(
			resource.GetAnnotations(),
			period,
			ptr.Deref(currentSuspend, false),
		))
		s.setSuspend(resource, ptr.To(!s.suspended))

	default:
		s.logger.Debug().Msgf("restoring: %s", resource.GetName())
		isAlreadyRestored, suspend, annotations, err := s.annotationMgr.RestoreBoolAnnotations(resource.GetAnnotations())
		if err != nil {
			return false, err
		}

		if isAlreadyRestored {
			return true, nil
		}

		s.setSuspend(resource, suspend)
		resource.SetAnnotations(annotations)
	}

	return false, nil
}
