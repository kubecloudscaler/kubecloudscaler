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

package hpa

import (
	"context"

	autoscaleV2 "k8s.io/api/autoscaling/v2"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
)

// hpaItem wraps autoscaleV2.HorizontalPodAutoscaler to implement ResourceItem interface.
type hpaItem struct {
	*autoscaleV2.HorizontalPodAutoscaler
}

func (h *hpaItem) GetName() string {
	return h.HorizontalPodAutoscaler.Name
}

func (h *hpaItem) GetNamespace() string {
	return h.HorizontalPodAutoscaler.Namespace
}

func (h *hpaItem) GetAnnotations() map[string]string {
	return h.HorizontalPodAutoscaler.Annotations
}

func (h *hpaItem) SetAnnotations(annotations map[string]string) {
	h.HorizontalPodAutoscaler.Annotations = annotations
}

// hpaLister implements ResourceLister for HPAs.
type hpaLister struct {
	client v2.AutoscalingV2Interface
}

func (l *hpaLister) List(ctx context.Context, namespace string, opts metaV1.ListOptions) ([]base.ResourceItem, error) {
	list, err := l.client.HorizontalPodAutoscalers(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	items := make([]base.ResourceItem, len(list.Items))
	for i := range list.Items {
		items[i] = &hpaItem{HorizontalPodAutoscaler: &list.Items[i]}
	}

	return items, nil
}

// hpaGetter implements ResourceGetter for HPAs.
type hpaGetter struct {
	client v2.AutoscalingV2Interface
}

func (g *hpaGetter) Get(ctx context.Context, namespace, name string, opts metaV1.GetOptions) (base.ResourceItem, error) {
	hpa, err := g.client.HorizontalPodAutoscalers(namespace).Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	return &hpaItem{HorizontalPodAutoscaler: hpa}, nil
}

// hpaUpdater implements ResourceUpdater for HPAs.
type hpaUpdater struct {
	client v2.AutoscalingV2Interface
}

func (u *hpaUpdater) Update(ctx context.Context, namespace string, resource base.ResourceItem, opts metaV1.UpdateOptions) (base.ResourceItem, error) {
	item, ok := resource.(*hpaItem)
	if !ok {
		return nil, &typeAssertionError{expected: "*hpaItem", got: resource}
	}

	updated, err := u.client.HorizontalPodAutoscalers(namespace).Update(ctx, item.HorizontalPodAutoscaler, opts)
	if err != nil {
		return nil, err
	}

	return &hpaItem{HorizontalPodAutoscaler: updated}, nil
}

// getMinMaxReplicas returns the min and max replicas from an HPA.
func getMinMaxReplicas(item base.ResourceItem) (*int32, *int32) {
	h, ok := item.(*hpaItem)
	if !ok {
		return nil, nil
	}
	return h.HorizontalPodAutoscaler.Spec.MinReplicas, &h.HorizontalPodAutoscaler.Spec.MaxReplicas
}

// setMinMaxReplicas sets the min and max replicas on an HPA.
func setMinMaxReplicas(item base.ResourceItem, minReplicas *int32, maxReplicas *int32) {
	h, ok := item.(*hpaItem)
	if !ok {
		return
	}
	h.HorizontalPodAutoscaler.Spec.MinReplicas = minReplicas
	if maxReplicas != nil {
		h.HorizontalPodAutoscaler.Spec.MaxReplicas = *maxReplicas
	}
}

type typeAssertionError struct {
	expected string
	got      interface{}
}

func (e *typeAssertionError) Error() string {
	return "type assertion failed"
}
