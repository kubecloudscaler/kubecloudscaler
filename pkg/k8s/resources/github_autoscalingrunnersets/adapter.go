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

package ars

import (
	"context"
	"fmt"

	actionsV1alpha1 "github.com/actions/actions-runner-controller/apis/actions.github.com/v1alpha1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
)

var runnerSetGVK = schema.GroupVersionKind{
	Group:   "actions.github.com",
	Version: "v1alpha1",
	Kind:    "AutoscalingRunnerSet",
}

// runnerSetItem wraps actionsV1alpha1.AutoscalingRunnerSet to implement ResourceItem interface.
type runnerSetItem struct {
	*actionsV1alpha1.AutoscalingRunnerSet
	unstructured *unstructured.Unstructured
}

func (r *runnerSetItem) GetName() string {
	return r.AutoscalingRunnerSet.Name
}

func (r *runnerSetItem) GetNamespace() string {
	return r.AutoscalingRunnerSet.Namespace
}

func (r *runnerSetItem) GetAnnotations() map[string]string {
	return r.AutoscalingRunnerSet.Annotations
}

func (r *runnerSetItem) SetAnnotations(annotations map[string]string) {
	r.AutoscalingRunnerSet.Annotations = annotations
}

// runnerSetLister implements ResourceLister for autoscaling runner sets.
type runnerSetLister struct {
	client dynamic.NamespaceableResourceInterface
}

func (l *runnerSetLister) List(ctx context.Context, namespace string, opts metaV1.ListOptions) ([]base.ResourceItem, error) {
	list, err := l.client.Namespace(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	items := make([]base.ResourceItem, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		runnerSet := &actionsV1alpha1.AutoscalingRunnerSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, runnerSet); err != nil {
			// Skip items that can't be converted
			continue
		}
		items = append(items, &runnerSetItem{
			AutoscalingRunnerSet: runnerSet,
			unstructured:         item,
		})
	}

	return items, nil
}

// runnerSetGetter implements ResourceGetter for autoscaling runner sets.
type runnerSetGetter struct {
	client dynamic.NamespaceableResourceInterface
}

func (g *runnerSetGetter) Get(ctx context.Context, namespace, name string, opts metaV1.GetOptions) (base.ResourceItem, error) {
	item, err := g.client.Namespace(namespace).Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	runnerSet := &actionsV1alpha1.AutoscalingRunnerSet{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, runnerSet); err != nil {
		return nil, fmt.Errorf("error converting unstructured to AutoscalingRunnerSet: %w", err)
	}

	return &runnerSetItem{
		AutoscalingRunnerSet: runnerSet,
		unstructured:         item,
	}, nil
}

// runnerSetUpdater implements ResourceUpdater for autoscaling runner sets.
type runnerSetUpdater struct {
	client dynamic.NamespaceableResourceInterface
}

func (u *runnerSetUpdater) Update(ctx context.Context, namespace string, resource base.ResourceItem, opts metaV1.UpdateOptions) (base.ResourceItem, error) {
	item, ok := resource.(*runnerSetItem)
	if !ok {
		return nil, &typeAssertionError{expected: "*runnerSetItem", got: resource}
	}

	// Set GVK before conversion
	item.AutoscalingRunnerSet.SetGroupVersionKind(runnerSetGVK)

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(item.AutoscalingRunnerSet)
	if err != nil {
		return nil, fmt.Errorf("error converting AutoscalingRunnerSet to unstructured: %w", err)
	}

	updated, err := u.client.Namespace(namespace).Update(
		ctx,
		&unstructured.Unstructured{Object: unstructuredObj},
		opts,
	)
	if err != nil {
		return nil, err
	}

	runnerSet := &actionsV1alpha1.AutoscalingRunnerSet{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(updated.Object, runnerSet); err != nil {
		return nil, fmt.Errorf("error converting updated unstructured to AutoscalingRunnerSet: %w", err)
	}

	return &runnerSetItem{
		AutoscalingRunnerSet: runnerSet,
		unstructured:         updated,
	}, nil
}

// getMinMaxReplicas returns the min and max replicas from an autoscaling runner set.
func getMinMaxReplicas(item base.ResourceItem) (*int32, *int32) {
	r, ok := item.(*runnerSetItem)
	if !ok {
		return nil, nil
	}
	minReplicas := intPtrToInt32Ptr(r.AutoscalingRunnerSet.Spec.MinRunners)
	maxReplicas := int32(ptr.Deref(r.AutoscalingRunnerSet.Spec.MaxRunners, 0))
	return minReplicas, &maxReplicas
}

// setMinMaxReplicas sets the min and max replicas on an autoscaling runner set.
func setMinMaxReplicas(item base.ResourceItem, minReplicas *int32, maxReplicas *int32) {
	r, ok := item.(*runnerSetItem)
	if !ok {
		return
	}
	r.AutoscalingRunnerSet.Spec.MinRunners = int32PtrToIntPtr(minReplicas)
	if maxReplicas != nil {
		r.AutoscalingRunnerSet.Spec.MaxRunners = int32ToIntPtr(*maxReplicas)
	}
}

// intPtrToInt32Ptr converts *int to *int32.
func intPtrToInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	v := int32(*value)
	return &v
}

// int32PtrToIntPtr converts *int32 to *int.
func int32PtrToIntPtr(value *int32) *int {
	if value == nil {
		return nil
	}
	v := int(*value)
	return &v
}

// int32ToIntPtr converts int32 to *int.
func int32ToIntPtr(value int32) *int {
	v := int(value)
	return &v
}

type typeAssertionError struct {
	expected string
	got      interface{}
}

func (e *typeAssertionError) Error() string {
	return fmt.Sprintf("type assertion failed: expected %s, got %T", e.expected, e.got)
}
