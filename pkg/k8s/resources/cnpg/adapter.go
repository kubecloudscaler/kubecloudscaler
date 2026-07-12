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

package cnpg

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
)

// clusterItem wraps a Cluster to implement the ResourceItem interface.
type clusterItem struct {
	*Cluster
	unstructured *unstructured.Unstructured
}

func (c *clusterItem) GetName() string {
	return c.Name
}

func (c *clusterItem) GetNamespace() string {
	return c.Namespace
}

func (c *clusterItem) GetAnnotations() map[string]string {
	return c.Annotations
}

func (c *clusterItem) SetAnnotations(annotations map[string]string) {
	c.Annotations = annotations
}

// clusterLister implements ResourceLister for CloudNativePG Clusters.
type clusterLister struct {
	client dynamic.NamespaceableResourceInterface
	logger *zerolog.Logger
}

func (l *clusterLister) List(ctx context.Context, namespace string, opts metaV1.ListOptions) ([]base.ResourceItem, error) {
	list, err := l.client.Namespace(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	items := make([]base.ResourceItem, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		cluster := &Cluster{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, cluster); err != nil {
			if l.logger != nil {
				l.logger.Warn().Err(err).
					Str("namespace", namespace).
					Str("name", item.GetName()).
					Msg("skipping Cluster item due to conversion failure")
			}
			continue
		}
		items = append(items, &clusterItem{
			Cluster:      cluster,
			unstructured: item,
		})
	}

	return items, nil
}

// clusterGetter implements ResourceGetter for CloudNativePG Clusters.
type clusterGetter struct {
	client dynamic.NamespaceableResourceInterface
}

func (g *clusterGetter) Get(ctx context.Context, namespace, name string, opts metaV1.GetOptions) (base.ResourceItem, error) {
	item, err := g.client.Namespace(namespace).Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	cluster := &Cluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, cluster); err != nil {
		return nil, fmt.Errorf("error converting unstructured to Cluster: %w", err)
	}

	return &clusterItem{
		Cluster:      cluster,
		unstructured: item,
	}, nil
}

// clusterUpdater implements ResourceUpdater for CloudNativePG Clusters.
type clusterUpdater struct {
	client dynamic.NamespaceableResourceInterface
}

func (u *clusterUpdater) Update(
	ctx context.Context,
	namespace string,
	resource base.ResourceItem,
	opts metaV1.UpdateOptions,
) (base.ResourceItem, error) {
	item, ok := resource.(*clusterItem)
	if !ok {
		return nil, base.NewTypeAssertionError("*clusterItem", resource)
	}

	// Start from a deep copy of the full live object so all Cluster fields not
	// mirrored in the local type (spec.instances, storage, etc.) are preserved.
	// Only the annotations we manage are patched.
	obj := item.unstructured.DeepCopy()
	obj.SetAnnotations(item.Cluster.GetAnnotations())

	updated, err := u.client.Namespace(namespace).Update(ctx, obj, opts)
	if err != nil {
		return nil, err
	}

	cluster := &Cluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(updated.Object, cluster); err != nil {
		return nil, fmt.Errorf("error converting updated unstructured to Cluster: %w", err)
	}

	return &clusterItem{
		Cluster:      cluster,
		unstructured: updated,
	}, nil
}
