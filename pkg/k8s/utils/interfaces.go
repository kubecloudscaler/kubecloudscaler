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

// Package utils provides interface definitions for Kubernetes resource management.
package utils

import (
	"context"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceLister defines the interface for listing namespaces
type NamespaceLister interface {
	List(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error)
}

// KubernetesClient defines the interface for Kubernetes operations
type KubernetesClient interface {
	CoreV1() CoreV1Interface
}

// CoreV1Interface defines the interface for Core V1 operations
type CoreV1Interface interface {
	Namespaces() NamespaceLister
}

// ConfigProvider defines the interface for configuration providers
type ConfigProvider interface {
	GetNamespaces() []string
	GetExcludeNamespaces() []string
	GetForceExcludeSystemNamespaces() bool
	GetLabelSelector() *metaV1.LabelSelector
	GetPeriod() interface{} // Using interface{} to avoid circular dependency
}

// NamespaceManager defines the interface for namespace management operations
type NamespaceManager interface {
	SetNamespaceList(ctx context.Context, config *Config) ([]string, error)
	PrepareSearch(ctx context.Context, config *Config) ([]string, metaV1.ListOptions, error)
	InitConfig(ctx context.Context, config *Config) (*K8sResource, error)
}

// AnnotationManager defines the interface for annotation management operations
//
//nolint:dupl // Mock implementation in mocks.go intentionally duplicates this interface structure
type AnnotationManager interface {
	AddAnnotations(annotations map[string]string, period interface{}) map[string]string
	RemoveAnnotations(annotations map[string]string) map[string]string
	AddMinMaxAnnotations(annot map[string]string, curPeriod interface{}, minReplicas *int32, max int32) map[string]string
	RestoreMinMaxAnnotations(annot map[string]string) (bool, *int32, int32, map[string]string, error)
	AddBoolAnnotations(annot map[string]string, curPeriod interface{}, value bool) map[string]string
	RestoreBoolAnnotations(annot map[string]string) (bool, *bool, map[string]string, error)
	AddIntAnnotations(annot map[string]string, curPeriod interface{}, value *int32) map[string]string
	RestoreIntAnnotations(annot map[string]string) (bool, *int32, map[string]string, error)
}

// Ensure that the concrete types implement the interfaces
var (
	_ NamespaceManager  = (*namespaceManager)(nil)
	_ AnnotationManager = (*annotationManager)(nil)
)
