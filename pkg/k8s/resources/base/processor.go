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
	"fmt"
	"slices"

	"github.com/rs/zerolog"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// ResourceLister defines the interface for listing resources.
type ResourceLister interface {
	List(ctx context.Context, namespace string, opts metaV1.ListOptions) ([]ResourceItem, error)
}

// ResourceGetter defines the interface for getting individual resources.
type ResourceGetter interface {
	Get(ctx context.Context, namespace, name string, opts metaV1.GetOptions) (ResourceItem, error)
}

// ResourceUpdater defines the interface for updating resources.
type ResourceUpdater interface {
	Update(ctx context.Context, namespace string, resource ResourceItem, opts metaV1.UpdateOptions) (ResourceItem, error)
}

// ResourceItem represents a Kubernetes resource item.
type ResourceItem interface {
	GetName() string
	GetNamespace() string
	GetAnnotations() map[string]string
	SetAnnotations(map[string]string)
}

// ScalingStrategy defines the interface for resource-specific scaling logic.
type ScalingStrategy interface {
	// ApplyScaling applies scaling logic based on period type.
	// Returns true if the resource was already restored (no action needed).
	ApplyScaling(ctx context.Context, resource ResourceItem, periodType string, period *periodPkg.Period) (bool, error)
	// GetKind returns the resource kind for status reporting.
	GetKind() string
}

// Processor handles the common resource scaling workflow.
type Processor struct {
	lister   ResourceLister
	getter   ResourceGetter
	updater  ResourceUpdater
	strategy ScalingStrategy
	resource *utils.K8sResource
	logger   *zerolog.Logger
}

// NewProcessor creates a new base processor.
func NewProcessor(
	lister ResourceLister,
	getter ResourceGetter,
	updater ResourceUpdater,
	strategy ScalingStrategy,
	resource *utils.K8sResource,
	logger *zerolog.Logger,
) *Processor {
	return &Processor{
		lister:   lister,
		getter:   getter,
		updater:  updater,
		strategy: strategy,
		resource: resource,
		logger:   logger,
	}
}

// ProcessResources processes all resources and returns success/failure results.
func (p *Processor) ProcessResources(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	scalerStatusSuccess := []common.ScalerStatusSuccess{}
	scalerStatusFailed := []common.ScalerStatusFailed{}

	// List all resources across namespaces
	list, err := p.listResources(ctx)
	if err != nil {
		return scalerStatusSuccess, scalerStatusFailed, err
	}

	p.logger.Debug().Msgf("number of %s: %d", p.strategy.GetKind(), len(list))

	// Process each resource
	for _, item := range list {
		if err := p.processResource(ctx, item, &scalerStatusSuccess, &scalerStatusFailed); err != nil {
			p.logger.Debug().Err(err).Str("resource", item.GetName()).Msg("error processing resource")
			// Continue processing other resources even if one fails
		}
	}

	return scalerStatusSuccess, scalerStatusFailed, nil
}

// listResources lists all resources across configured namespaces.
func (p *Processor) listResources(ctx context.Context) ([]ResourceItem, error) {
	var allItems []ResourceItem

	for _, ns := range p.resource.NsList {
		p.logger.Debug().Msgf("found namespace: %s", ns)

		items, err := p.lister.List(ctx, ns, p.resource.ListOptions)
		if err != nil {
			p.logger.Debug().Err(err).Msgf("error listing %s", p.strategy.GetKind())
			// Use plural form for error message to match existing test expectations
			kindPlural := p.strategy.GetKind() + "s"
			return nil, fmt.Errorf("error listing %s: %w", kindPlural, err)
		}

		allItems = append(allItems, items...)
	}

	// Filter by resource names if specified
	if len(p.resource.Names) > 0 {
		allItems = slices.DeleteFunc(allItems, func(item ResourceItem) bool {
			return !slices.Contains(p.resource.Names, item.GetName())
		})
	}

	return allItems, nil
}

// processResource processes a single resource.
func (p *Processor) processResource(
	ctx context.Context,
	item ResourceItem,
	successList *[]common.ScalerStatusSuccess,
	failedList *[]common.ScalerStatusFailed,
) error {
	p.logger.Debug().Msgf("resource-name: %s", item.GetName())

	// Get the full resource
	resource, err := p.getter.Get(ctx, item.GetNamespace(), item.GetName(), metaV1.GetOptions{})
	if err != nil {
		p.appendFailure(failedList, item.GetName(), err.Error())
		return err
	}

	// Apply scaling strategy
	alreadyRestored, err := p.strategy.ApplyScaling(ctx, resource, p.resource.Period.Type, p.resource.Period)
	if err != nil {
		p.appendFailure(failedList, item.GetName(), err.Error())
		return err
	}

	if alreadyRestored {
		p.logger.Debug().Msgf("nothing to do: %s", item.GetName())
		return nil
	}

	// Update the resource
	p.logger.Debug().Msgf("update %s: %s", p.strategy.GetKind(), item.GetName())

	_, err = p.updater.Update(ctx, resource.GetNamespace(), resource, metaV1.UpdateOptions{
		FieldManager: utils.FieldManager,
	})
	if err != nil {
		p.appendFailure(failedList, item.GetName(), err.Error())
		return err
	}

	// Record success
	p.appendSuccess(successList, item.GetName())
	return nil
}

// appendFailure appends a failure to the list.
func (p *Processor) appendFailure(failedList *[]common.ScalerStatusFailed, name, reason string) {
	*failedList = append(*failedList, common.ScalerStatusFailed{
		Kind:   p.strategy.GetKind(),
		Name:   name,
		Reason: reason,
	})
}

// appendSuccess appends a success to the list.
func (p *Processor) appendSuccess(successList *[]common.ScalerStatusSuccess, name string) {
	*successList = append(*successList, common.ScalerStatusSuccess{
		Kind: p.strategy.GetKind(),
		Name: name,
	})
}
