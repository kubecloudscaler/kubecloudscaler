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

// Package utils provides adapter implementations for Kubernetes resource management.
package utils

import (
	"context"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// KubernetesClientAdapter adapts kubernetes.Interface to our KubernetesClient interface
type KubernetesClientAdapter struct {
	client kubernetes.Interface
}

// NewKubernetesClientAdapter creates a new adapter
func NewKubernetesClientAdapter(client kubernetes.Interface) KubernetesClient {
	return &KubernetesClientAdapter{client: client}
}

// CoreV1 returns the CoreV1Interface
func (a *KubernetesClientAdapter) CoreV1() CoreV1Interface {
	return &CoreV1InterfaceAdapter{client: a.client.CoreV1()}
}

// CoreV1InterfaceAdapter adapts kubernetes.CoreV1Interface to our CoreV1Interface
type CoreV1InterfaceAdapter struct {
	client corev1.CoreV1Interface
}

// Namespaces returns the NamespaceLister
func (a *CoreV1InterfaceAdapter) Namespaces() NamespaceLister {
	return &NamespaceListerAdapter{client: a.client.Namespaces()}
}

// NamespaceListerAdapter adapts kubernetes.NamespaceInterface to our NamespaceLister
type NamespaceListerAdapter struct {
	client corev1.NamespaceInterface
}

// List lists namespaces
//
//nolint:gocritic // metav1.ListOptions is a Kubernetes API type, passing by value is idiomatic
func (a *NamespaceListerAdapter) List(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
	return a.client.List(ctx, opts)
}

// NewFakeKubernetesClient creates a fake Kubernetes client that implements our interfaces
func NewFakeKubernetesClient(objects ...interface{}) KubernetesClient {
	// Convert interface{} to runtime.Object
	runtimeObjects := make([]interface{}, len(objects))
	copy(runtimeObjects, objects)
	fakeClient := fake.NewSimpleClientset()
	return NewKubernetesClientAdapter(fakeClient)
}
