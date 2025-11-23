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

package cronjobs

import (
	"context"
	"fmt"

	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
)

// cronJobItem wraps batchV1.CronJob to implement ResourceItem interface.
type cronJobItem struct {
	*batchV1.CronJob
}

func (c *cronJobItem) GetName() string {
	return c.CronJob.Name
}

func (c *cronJobItem) GetNamespace() string {
	return c.CronJob.Namespace
}

func (c *cronJobItem) GetAnnotations() map[string]string {
	return c.CronJob.Annotations
}

func (c *cronJobItem) SetAnnotations(annotations map[string]string) {
	c.CronJob.Annotations = annotations
}

// cronJobLister implements ResourceLister for cronjobs.
type cronJobLister struct {
	client v1.BatchV1Interface
}

//nolint:gocritic // hugeParam: Kubernetes API types are passed by value for immutability
func (l *cronJobLister) List(ctx context.Context, namespace string, opts metaV1.ListOptions) ([]base.ResourceItem, error) {
	list, err := l.client.CronJobs(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	items := make([]base.ResourceItem, len(list.Items))
	for i := range list.Items {
		items[i] = &cronJobItem{CronJob: &list.Items[i]}
	}

	return items, nil
}

// cronJobGetter implements ResourceGetter for cronjobs.
type cronJobGetter struct {
	client v1.BatchV1Interface
}

func (g *cronJobGetter) Get(ctx context.Context, namespace, name string, opts metaV1.GetOptions) (base.ResourceItem, error) {
	cronjob, err := g.client.CronJobs(namespace).Get(ctx, name, opts)
	if err != nil {
		return nil, err
	}

	return &cronJobItem{CronJob: cronjob}, nil
}

// cronJobUpdater implements ResourceUpdater for cronjobs.
type cronJobUpdater struct {
	client v1.BatchV1Interface
}

//nolint:gocritic // hugeParam: Kubernetes API types are passed by value for immutability
func (u *cronJobUpdater) Update(ctx context.Context, namespace string, resource base.ResourceItem, opts metaV1.UpdateOptions) (base.ResourceItem, error) {
	item, ok := resource.(*cronJobItem)
	if !ok {
		return nil, &typeAssertionError{expected: "*cronJobItem", got: resource}
	}

	updated, err := u.client.CronJobs(namespace).Update(ctx, item.CronJob, opts)
	if err != nil {
		return nil, err
	}

	return &cronJobItem{CronJob: updated}, nil
}

// getSuspend returns the suspend value from a cronjob.
func getSuspend(item base.ResourceItem) *bool {
	c, ok := item.(*cronJobItem)
	if !ok {
		return nil
	}
	return c.CronJob.Spec.Suspend
}

// setSuspend sets the suspend value on a cronjob.
func setSuspend(item base.ResourceItem, suspend *bool) {
	c, ok := item.(*cronJobItem)
	if !ok {
		return
	}
	c.CronJob.Spec.Suspend = suspend
}

// onUpError returns an error when trying to scale up a cronjob (not supported).
func onUpError(item base.ResourceItem) error {
	return fmt.Errorf("cronjob can only be scaled down")
}

type typeAssertionError struct {
	expected string
	got      interface{}
}

func (e *typeAssertionError) Error() string {
	return fmt.Sprintf("type assertion failed: expected %s, got %T", e.expected, e.got)
}
