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

package utils

import (
	"context"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// kubernetesClientAdapter adapts kubernetes.Interface to KubernetesClient interface.
type kubernetesClientAdapter struct {
	client kubernetes.Interface
}

// coreV1InterfaceAdapter adapts k8s CoreV1Interface to utils.CoreV1Interface.
type coreV1InterfaceAdapter struct {
	client corev1.CoreV1Interface
}

// namespaceListerAdapter adapts k8s NamespaceInterface to utils.NamespaceLister.
type namespaceListerAdapter struct {
	lister corev1.NamespaceInterface
}

// NewKubernetesClientAdapter creates an adapter that wraps kubernetes.Interface to implement KubernetesClient.
func NewKubernetesClientAdapter(client kubernetes.Interface) KubernetesClient {
	return &kubernetesClientAdapter{client: client}
}

// CoreV1 returns the CoreV1Interface adapter.
func (k *kubernetesClientAdapter) CoreV1() CoreV1Interface {
	return &coreV1InterfaceAdapter{client: k.client.CoreV1()}
}

// Namespaces returns the NamespaceLister adapter.
func (c *coreV1InterfaceAdapter) Namespaces() NamespaceLister {
	return &namespaceListerAdapter{lister: c.client.Namespaces()}
}

// List lists namespaces.
func (n *namespaceListerAdapter) List(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
	return n.lister.List(ctx, opts)
}

// NewFakeKubernetesClient creates a fake Kubernetes client that implements our interfaces.
func NewFakeKubernetesClient(objects ...interface{}) KubernetesClient {
	// Convert interface{} to runtime.Object
	runtimeObjects := make([]runtime.Object, 0, len(objects))
	for _, obj := range objects {
		if runtimeObj, ok := obj.(runtime.Object); ok {
			runtimeObjects = append(runtimeObjects, runtimeObj)
		}
	}
	fakeClient := fake.NewSimpleClientset(runtimeObjects...)
	return NewKubernetesClientAdapter(fakeClient)
}
