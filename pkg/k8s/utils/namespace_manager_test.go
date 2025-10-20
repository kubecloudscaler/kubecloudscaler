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
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespaceManagerOriginal(t *testing.T) {
	RegisterFailHandler(Fail)
	// RunSpecs is handled by suite_test.go
}

var _ = Describe("NamespaceManager", func() {
	var (
		ctx          context.Context
		logger       zerolog.Logger
		namespaceMgr NamespaceManager
		mockClient   *MockKubernetesClient
		config       *Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = zerolog.Nop()
		mockClient = &MockKubernetesClient{}
		namespaceMgr = NewNamespaceManager(mockClient, logger)
		config = &Config{
			ForceExcludeSystemNamespaces: true,
		}
	})

	Context("SetNamespaceList", func() {
		It("should return specified namespaces when provided", func() {
			config.Namespaces = []string{"namespace1", "namespace2", "namespace3"}

			nsList, err := namespaceMgr.SetNamespaceList(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(Equal([]string{"namespace1", "namespace2", "namespace3"}))
		})

		It("should return namespaces from cluster when no namespaces specified", func() {
			config.Namespaces = []string{}
			mockClient.CoreV1Func = func() CoreV1Interface {
				return &MockCoreV1Interface{
					NamespacesFunc: func() NamespaceLister {
						return &MockNamespaceLister{
							ListFunc: func(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
								return &coreV1.NamespaceList{
									Items: []coreV1.Namespace{
										{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-1"}},
										{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-2"}},
										{ObjectMeta: metaV1.ObjectMeta{Name: "kube-system"}},
									},
								}, nil
							},
						}
					},
				}
			}

			nsList, err := namespaceMgr.SetNamespaceList(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElements("test-ns-1", "test-ns-2"))
			Expect(nsList).ToNot(ContainElement("kube-system"))
		})

		It("should exclude specified namespaces", func() {
			config.Namespaces = []string{}
			config.ExcludeNamespaces = []string{"exclude-ns"}
			mockClient.CoreV1Func = func() CoreV1Interface {
				return &MockCoreV1Interface{
					NamespacesFunc: func() NamespaceLister {
						return &MockNamespaceLister{
							ListFunc: func(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
								return &coreV1.NamespaceList{
									Items: []coreV1.Namespace{
										{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-1"}},
										{ObjectMeta: metaV1.ObjectMeta{Name: "exclude-ns"}},
										{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-2"}},
									},
								}, nil
							},
						}
					},
				}
			}

			nsList, err := namespaceMgr.SetNamespaceList(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElements("test-ns-1", "test-ns-2"))
			Expect(nsList).ToNot(ContainElement("exclude-ns"))
		})

		It("should return error when client fails to list namespaces", func() {
			config.Namespaces = []string{}
			mockClient.CoreV1Func = func() CoreV1Interface {
				return &MockCoreV1Interface{
					NamespacesFunc: func() NamespaceLister {
						return &MockNamespaceLister{
							ListFunc: func(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
								return nil, errors.New("mock error")
							},
						}
					},
				}
			}

			nsList, err := namespaceMgr.SetNamespaceList(ctx, config)

			Expect(err).To(HaveOccurred())
			Expect(nsList).To(BeEmpty())
			Expect(err.Error()).To(ContainSubstring("error listing namespaces"))
		})
	})

	Context("PrepareSearch", func() {
		It("should prepare search with default label selector", func() {
			config.Namespaces = []string{"test-ns"}
			mockClient.CoreV1Func = func() CoreV1Interface {
				return &MockCoreV1Interface{
					NamespacesFunc: func() NamespaceLister {
						return &MockNamespaceLister{
							ListFunc: func(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
								return &coreV1.NamespaceList{
									Items: []coreV1.Namespace{
										{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns"}},
									},
								}, nil
							},
						}
					},
				}
			}

			nsList, listOptions, err := namespaceMgr.PrepareSearch(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElement("test-ns"))
			Expect(listOptions.LabelSelector).To(ContainSubstring("kubecloudscaler.cloud/ignore"))
		})

		It("should merge custom label selector with default", func() {
			config.Namespaces = []string{"test-ns"}
			config.LabelSelector = &metaV1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
				MatchExpressions: []metaV1.LabelSelectorRequirement{
					{
						Key:      "environment",
						Operator: metaV1.LabelSelectorOpIn,
						Values:   []string{"prod", "staging"},
					},
				},
			}

			nsList, listOptions, err := namespaceMgr.PrepareSearch(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElement("test-ns"))
			Expect(listOptions.LabelSelector).To(ContainSubstring("app=test"))
			Expect(listOptions.LabelSelector).To(ContainSubstring("environment in (prod,staging)"))
			Expect(listOptions.LabelSelector).To(ContainSubstring("kubecloudscaler.cloud/ignore"))
		})
	})

	Context("InitConfig", func() {
		It("should initialize K8sResource successfully", func() {
			config.Namespaces = []string{"test-ns"}
			mockClient.CoreV1Func = func() CoreV1Interface {
				return &MockCoreV1Interface{
					NamespacesFunc: func() NamespaceLister {
						return &MockNamespaceLister{
							ListFunc: func(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
								return &coreV1.NamespaceList{
									Items: []coreV1.Namespace{
										{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns"}},
									},
								}, nil
							},
						}
					},
				}
			}

			resource, err := namespaceMgr.InitConfig(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(resource).ToNot(BeNil())
			Expect(resource.Period).To(Equal(config.Period))
			Expect(resource.NsList).To(ContainElement("test-ns"))
			Expect(resource.ListOptions.LabelSelector).ToNot(BeEmpty())
		})
	})
})
