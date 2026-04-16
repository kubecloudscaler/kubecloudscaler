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

package scaledobjects

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
)

// scaledObjectItem wraps a ScaledObject to implement ResourceItem interface.
type scaledObjectItem struct {
	*ScaledObject
	unstructured *unstructured.Unstructured
}

func (s *scaledObjectItem) GetName() string {
	return s.ScaledObject.Name
}

func (s *scaledObjectItem) GetNamespace() string {
	return s.ScaledObject.Namespace
}

func (s *scaledObjectItem) GetAnnotations() map[string]string {
	return s.ScaledObject.Annotations
}

func (s *scaledObjectItem) SetAnnotations(annotations map[string]string) {
	s.ScaledObject.Annotations = annotations
}

// scaledObjectLister implements ResourceLister for KEDA ScaledObjects.
type scaledObjectLister struct {
	client dynamic.NamespaceableResourceInterface
	logger *zerolog.Logger
}

func (l *scaledObjectLister) List(ctx context.Context, namespace string, opts metaV1.ListOptions) ([]base.ResourceItem, error) {
	list, err := l.client.Namespace(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	items := make([]base.ResourceItem, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		so := &ScaledObject{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, so); err != nil {
			if l.logger != nil {
				l.logger.Warn().Err(err).
					Str("namespace", namespace).
					Str("name", item.GetName()).
					Msg("skipping ScaledObject item due to conversion failure")
			}
			continue
		}
		items = append(items, &scaledObjectItem{
			ScaledObject: so,
			unstructured: item,
		})
	}

	return items, nil
}

// scaledObjectGetter implements ResourceGetter for KEDA ScaledObjects.
type scaledObjectGetter struct {
	client dynamic.NamespaceableResourceInterface
}

func (g *scaledObjectGetter) Get(ctx context.Context, namespace, name string, opts metaV1.GetOptions) (base.ResourceItem, error) {
	item, err := g.client.Namespace(namespace).Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	so := &ScaledObject{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, so); err != nil {
		return nil, fmt.Errorf("error converting unstructured to ScaledObject: %w", err)
	}

	return &scaledObjectItem{
		ScaledObject: so,
		unstructured: item,
	}, nil
}

// scaledObjectUpdater implements ResourceUpdater for KEDA ScaledObjects.
type scaledObjectUpdater struct {
	client dynamic.NamespaceableResourceInterface
}

func (u *scaledObjectUpdater) Update(ctx context.Context, namespace string, resource base.ResourceItem, opts metaV1.UpdateOptions) (base.ResourceItem, error) {
	item, ok := resource.(*scaledObjectItem)
	if !ok {
		return nil, base.NewTypeAssertionError("*scaledObjectItem", resource)
	}

	// Start from a deep copy of the full live object so all KEDA fields not
	// mirrored in the local ScaledObject type (scaleTargetRef, triggers, etc.)
	// are preserved. Only patch the fields we manage.
	obj := item.unstructured.DeepCopy()
	obj.SetAnnotations(item.ScaledObject.GetAnnotations())
	if err := syncSpecReplicas(item.ScaledObject.Spec, obj); err != nil {
		return nil, fmt.Errorf("error syncing spec replicas to unstructured: %w", err)
	}

	updated, err := u.client.Namespace(namespace).Update(ctx, obj, opts)
	if err != nil {
		return nil, err
	}

	so := &ScaledObject{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(updated.Object, so); err != nil {
		return nil, fmt.Errorf("error converting updated unstructured to ScaledObject: %w", err)
	}

	return &scaledObjectItem{
		ScaledObject: so,
		unstructured: updated,
	}, nil
}

// syncSpecReplicas patches only the minReplicaCount and maxReplicaCount fields
// in an unstructured ScaledObject, leaving all other spec fields untouched.
func syncSpecReplicas(spec ScaledObjectSpec, obj *unstructured.Unstructured) error {
	if spec.MinReplicaCount != nil {
		if err := unstructured.SetNestedField(obj.Object, int64(*spec.MinReplicaCount), "spec", "minReplicaCount"); err != nil {
			return err
		}
	} else {
		unstructured.RemoveNestedField(obj.Object, "spec", "minReplicaCount")
	}
	if spec.MaxReplicaCount != nil {
		if err := unstructured.SetNestedField(obj.Object, int64(*spec.MaxReplicaCount), "spec", "maxReplicaCount"); err != nil {
			return err
		}
	} else {
		unstructured.RemoveNestedField(obj.Object, "spec", "maxReplicaCount")
	}
	return nil
}

// getMinMaxReplicas returns the min and max replicas from a ScaledObject.
func getMinMaxReplicas(item base.ResourceItem) (*int32, *int32) {
	s, ok := item.(*scaledObjectItem)
	if !ok {
		return nil, nil
	}
	return s.ScaledObject.Spec.MinReplicaCount, s.ScaledObject.Spec.MaxReplicaCount
}

// setMinMaxReplicas sets the min and max replicas on a ScaledObject.
func setMinMaxReplicas(item base.ResourceItem, minReplicas, maxReplicas *int32) {
	s, ok := item.(*scaledObjectItem)
	if !ok {
		return
	}
	s.ScaledObject.Spec.MinReplicaCount = ptr.To(ptr.Deref(minReplicas, 0))
	if maxReplicas != nil {
		s.ScaledObject.Spec.MaxReplicaCount = ptr.To(*maxReplicas)
	}
}
