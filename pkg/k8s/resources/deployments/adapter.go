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

package deployments

import (
	"context"

	appsV1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
)

// deploymentItem wraps appsV1.Deployment to implement ResourceItem interface.
type deploymentItem struct {
	*appsV1.Deployment
}

func (d *deploymentItem) GetName() string {
	return d.Deployment.Name
}

func (d *deploymentItem) GetNamespace() string {
	return d.Deployment.Namespace
}

func (d *deploymentItem) GetAnnotations() map[string]string {
	return d.Deployment.Annotations
}

func (d *deploymentItem) SetAnnotations(annotations map[string]string) {
	d.Deployment.Annotations = annotations
}

// deploymentLister implements ResourceLister for deployments.
type deploymentLister struct {
	client v1.AppsV1Interface
}

//nolint:gocritic // hugeParam: Kubernetes API types are passed by value for immutability
func (l *deploymentLister) List(ctx context.Context, namespace string, opts metaV1.ListOptions) ([]base.ResourceItem, error) {
	list, err := l.client.Deployments(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	items := make([]base.ResourceItem, len(list.Items))
	for i := range list.Items {
		items[i] = &deploymentItem{Deployment: &list.Items[i]}
	}

	return items, nil
}

// deploymentGetter implements ResourceGetter for deployments.
type deploymentGetter struct {
	client v1.AppsV1Interface
}

func (g *deploymentGetter) Get(ctx context.Context, namespace, name string, opts metaV1.GetOptions) (base.ResourceItem, error) {
	deploy, err := g.client.Deployments(namespace).Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	return &deploymentItem{Deployment: deploy}, nil
}

// deploymentUpdater implements ResourceUpdater for deployments.
type deploymentUpdater struct {
	client v1.AppsV1Interface
}

//nolint:gocritic // hugeParam: Kubernetes API types are passed by value for immutability
func (u *deploymentUpdater) Update(ctx context.Context, namespace string, resource base.ResourceItem, opts metaV1.UpdateOptions) (base.ResourceItem, error) {
	item, ok := resource.(*deploymentItem)
	if !ok {
		return nil, &typeAssertionError{expected: "*deploymentItem", got: resource}
	}

	updated, err := u.client.Deployments(namespace).Update(ctx, item.Deployment, opts)
	if err != nil {
		return nil, err
	}

	return &deploymentItem{Deployment: updated}, nil
}

// getReplicas returns the replicas from a deployment.
func getReplicas(item base.ResourceItem) *int32 {
	d, ok := item.(*deploymentItem)
	if !ok {
		return nil
	}
	return d.Deployment.Spec.Replicas
}

// setReplicas sets the replicas on a deployment.
func setReplicas(item base.ResourceItem, replicas *int32) {
	d, ok := item.(*deploymentItem)
	if !ok {
		return
	}
	d.Deployment.Spec.Replicas = replicas
}

type typeAssertionError struct {
	expected string
	got      interface{}
}

func (e *typeAssertionError) Error() string {
	return "type assertion failed"
}
