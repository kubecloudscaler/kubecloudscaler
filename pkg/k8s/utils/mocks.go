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
)

// MockKubernetesClient is a mock implementation of KubernetesClient
type MockKubernetesClient struct {
	CoreV1Func func() CoreV1Interface
}

func (m *MockKubernetesClient) CoreV1() CoreV1Interface {
	if m.CoreV1Func != nil {
		return m.CoreV1Func()
	}
	return &MockCoreV1Interface{}
}

// MockCoreV1Interface is a mock implementation of CoreV1Interface
type MockCoreV1Interface struct {
	NamespacesFunc func() NamespaceLister
}

func (m *MockCoreV1Interface) Namespaces() NamespaceLister {
	if m.NamespacesFunc != nil {
		return m.NamespacesFunc()
	}
	return &MockNamespaceLister{}
}

// MockNamespaceLister is a mock implementation of NamespaceLister
type MockNamespaceLister struct {
	ListFunc func(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error)
}

func (m *MockNamespaceLister) List(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, opts)
	}
	return &coreV1.NamespaceList{}, nil
}

// MockConfigProvider is a mock implementation of ConfigProvider
type MockConfigProvider struct {
	GetNamespacesFunc                   func() []string
	GetExcludeNamespacesFunc            func() []string
	GetForceExcludeSystemNamespacesFunc func() bool
	GetLabelSelectorFunc                func() *metaV1.LabelSelector
	GetPeriodFunc                       func() interface{}
}

func (m *MockConfigProvider) GetNamespaces() []string {
	if m.GetNamespacesFunc != nil {
		return m.GetNamespacesFunc()
	}
	return []string{}
}

func (m *MockConfigProvider) GetExcludeNamespaces() []string {
	if m.GetExcludeNamespacesFunc != nil {
		return m.GetExcludeNamespacesFunc()
	}
	return []string{}
}

func (m *MockConfigProvider) GetForceExcludeSystemNamespaces() bool {
	if m.GetForceExcludeSystemNamespacesFunc != nil {
		return m.GetForceExcludeSystemNamespacesFunc()
	}
	return false
}

func (m *MockConfigProvider) GetLabelSelector() *metaV1.LabelSelector {
	if m.GetLabelSelectorFunc != nil {
		return m.GetLabelSelectorFunc()
	}
	return nil
}

func (m *MockConfigProvider) GetPeriod() interface{} {
	if m.GetPeriodFunc != nil {
		return m.GetPeriodFunc()
	}
	return nil
}

// MockAnnotationManager is a mock implementation of AnnotationManager
type MockAnnotationManager struct {
	AddAnnotationsFunc           func(annotations map[string]string, period interface{}) map[string]string
	RemoveAnnotationsFunc        func(annotations map[string]string) map[string]string
	AddMinMaxAnnotationsFunc     func(annot map[string]string, curPeriod interface{}, min *int32, max int32) map[string]string
	RestoreMinMaxAnnotationsFunc func(annot map[string]string) (bool, *int32, int32, map[string]string, error)
	AddBoolAnnotationsFunc       func(annot map[string]string, curPeriod interface{}, value bool) map[string]string
	RestoreBoolAnnotationsFunc   func(annot map[string]string) (bool, *bool, map[string]string, error)
	AddIntAnnotationsFunc        func(annot map[string]string, curPeriod interface{}, value *int32) map[string]string
	RestoreIntAnnotationsFunc    func(annot map[string]string) (bool, *int32, map[string]string, error)
}

func (m *MockAnnotationManager) AddAnnotations(annotations map[string]string, period interface{}) map[string]string {
	if m.AddAnnotationsFunc != nil {
		return m.AddAnnotationsFunc(annotations, period)
	}
	return annotations
}

func (m *MockAnnotationManager) RemoveAnnotations(annotations map[string]string) map[string]string {
	if m.RemoveAnnotationsFunc != nil {
		return m.RemoveAnnotationsFunc(annotations)
	}
	return annotations
}

func (m *MockAnnotationManager) AddMinMaxAnnotations(annot map[string]string, curPeriod interface{}, min *int32, max int32) map[string]string {
	if m.AddMinMaxAnnotationsFunc != nil {
		return m.AddMinMaxAnnotationsFunc(annot, curPeriod, min, max)
	}
	return annot
}

func (m *MockAnnotationManager) RestoreMinMaxAnnotations(annot map[string]string) (bool, *int32, int32, map[string]string, error) {
	if m.RestoreMinMaxAnnotationsFunc != nil {
		return m.RestoreMinMaxAnnotationsFunc(annot)
	}
	return true, nil, 0, annot, nil
}

func (m *MockAnnotationManager) AddBoolAnnotations(annot map[string]string, curPeriod interface{}, value bool) map[string]string {
	if m.AddBoolAnnotationsFunc != nil {
		return m.AddBoolAnnotationsFunc(annot, curPeriod, value)
	}
	return annot
}

func (m *MockAnnotationManager) RestoreBoolAnnotations(annot map[string]string) (bool, *bool, map[string]string, error) {
	if m.RestoreBoolAnnotationsFunc != nil {
		return m.RestoreBoolAnnotationsFunc(annot)
	}
	return true, nil, annot, nil
}

func (m *MockAnnotationManager) AddIntAnnotations(annot map[string]string, curPeriod interface{}, value *int32) map[string]string {
	if m.AddIntAnnotationsFunc != nil {
		return m.AddIntAnnotationsFunc(annot, curPeriod, value)
	}
	return annot
}

func (m *MockAnnotationManager) RestoreIntAnnotations(annot map[string]string) (bool, *int32, map[string]string, error) {
	if m.RestoreIntAnnotationsFunc != nil {
		return m.RestoreIntAnnotationsFunc(annot)
	}
	return true, nil, annot, nil
}

// Helper functions for creating test data

// NewMockKubernetesClientWithNamespaces creates a mock client that returns the specified namespaces
func NewMockKubernetesClientWithNamespaces(namespaces []string) *MockKubernetesClient {
	return &MockKubernetesClient{
		CoreV1Func: func() CoreV1Interface {
			return &MockCoreV1Interface{
				NamespacesFunc: func() NamespaceLister {
					return &MockNamespaceLister{
						ListFunc: func(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
							items := make([]coreV1.Namespace, len(namespaces))
							for i, ns := range namespaces {
								items[i] = coreV1.Namespace{
									ObjectMeta: metaV1.ObjectMeta{Name: ns},
								}
							}
							return &coreV1.NamespaceList{Items: items}, nil
						},
					}
				},
			}
		},
	}
}

// NewMockKubernetesClientWithError creates a mock client that returns an error
func NewMockKubernetesClientWithError(err error) *MockKubernetesClient {
	return &MockKubernetesClient{
		CoreV1Func: func() CoreV1Interface {
			return &MockCoreV1Interface{
				NamespacesFunc: func() NamespaceLister {
					return &MockNamespaceLister{
						ListFunc: func(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
							return nil, err
						},
					}
				},
			}
		},
	}
}

// NewMockConfigProvider creates a mock config provider with the specified values
func NewMockConfigProvider(namespaces []string, excludeNamespaces []string, forceExcludeSystem bool) *MockConfigProvider {
	return &MockConfigProvider{
		GetNamespacesFunc: func() []string {
			return namespaces
		},
		GetExcludeNamespacesFunc: func() []string {
			return excludeNamespaces
		},
		GetForceExcludeSystemNamespacesFunc: func() bool {
			return forceExcludeSystem
		},
	}
}
