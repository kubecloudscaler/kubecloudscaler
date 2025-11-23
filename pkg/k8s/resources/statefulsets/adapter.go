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

package statefulsets

import (
	"context"

	appsV1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
)

// statefulSetItem wraps appsV1.StatefulSet to implement ResourceItem interface.
type statefulSetItem struct {
	*appsV1.StatefulSet
}

func (s *statefulSetItem) GetName() string {
	return s.StatefulSet.Name
}

func (s *statefulSetItem) GetNamespace() string {
	return s.StatefulSet.Namespace
}

func (s *statefulSetItem) GetAnnotations() map[string]string {
	return s.StatefulSet.Annotations
}

func (s *statefulSetItem) SetAnnotations(annotations map[string]string) {
	s.StatefulSet.Annotations = annotations
}

// statefulSetLister implements ResourceLister for statefulsets.
type statefulSetLister struct {
	client v1.AppsV1Interface
}

func (l *statefulSetLister) List(ctx context.Context, namespace string, opts metaV1.ListOptions) ([]base.ResourceItem, error) {
	list, err := l.client.StatefulSets(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	items := make([]base.ResourceItem, len(list.Items))
	for i := range list.Items {
		items[i] = &statefulSetItem{StatefulSet: &list.Items[i]}
	}

	return items, nil
}

// statefulSetGetter implements ResourceGetter for statefulsets.
type statefulSetGetter struct {
	client v1.AppsV1Interface
}

func (g *statefulSetGetter) Get(ctx context.Context, namespace, name string, opts metaV1.GetOptions) (base.ResourceItem, error) {
	stateful, err := g.client.StatefulSets(namespace).Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	return &statefulSetItem{StatefulSet: stateful}, nil
}

// statefulSetUpdater implements ResourceUpdater for statefulsets.
type statefulSetUpdater struct {
	client v1.AppsV1Interface
}

func (u *statefulSetUpdater) Update(ctx context.Context, namespace string, resource base.ResourceItem, opts metaV1.UpdateOptions) (base.ResourceItem, error) {
	item, ok := resource.(*statefulSetItem)
	if !ok {
		return nil, &typeAssertionError{expected: "*statefulSetItem", got: resource}
	}

	updated, err := u.client.StatefulSets(namespace).Update(ctx, item.StatefulSet, opts)
	if err != nil {
		return nil, err
	}

	return &statefulSetItem{StatefulSet: updated}, nil
}

// getReplicas returns the replicas from a statefulset.
func getReplicas(item base.ResourceItem) *int32 {
	s, ok := item.(*statefulSetItem)
	if !ok {
		return nil
	}
	return s.StatefulSet.Spec.Replicas
}

// setReplicas sets the replicas on a statefulset.
func setReplicas(item base.ResourceItem, replicas *int32) {
	s, ok := item.(*statefulSetItem)
	if !ok {
		return
	}
	s.StatefulSet.Spec.Replicas = replicas
}

type typeAssertionError struct {
	expected string
	got      interface{}
}

func (e *typeAssertionError) Error() string {
	return "type assertion failed"
}
