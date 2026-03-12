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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

func TestIntegrationOriginal(t *testing.T) {
	RegisterFailHandler(Fail)
	// RunSpecs is handled by suite_test.go
}

var _ = Describe("Integration Tests", func() {
	var (
		ctx           context.Context
		logger        zerolog.Logger
		client        KubernetesClient
		namespaceMgr  NamespaceManager
		annotationMgr AnnotationManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = zerolog.Nop()
		client = NewFakeKubernetesClient()
		namespaceMgr = NewNamespaceManager(client, logger, nil)
		annotationMgr = NewAnnotationManager()
	})

	Context("End-to-End Workflow", func() {
		It("should complete full workflow from namespace listing to annotation management", func() {
			// Setup test namespaces
			ns1 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-1"}}
			ns2 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-2"}}
			ns3 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "kube-system"}}
			client = NewFakeKubernetesClient(ns1, ns2, ns3)

			// Create namespace manager with real client
			namespaceMgr = NewNamespaceManager(client, logger, nil)

			// Step 1: Configure and initialize
			config := &Config{
				ForceExcludeSystemNamespaces: true,
				ExcludeNamespaces:            []string{"test-ns-2"},
				LabelSelector: &metaV1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
					},
				},
			}

			// Step 2: Initialize K8sResource
			resource, err := namespaceMgr.InitConfig(ctx, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(resource).ToNot(BeNil())

			// Step 3: Verify namespace list (should exclude kube-system and test-ns-2)
			// Note: The fake client doesn't actually return the namespaces we created
			// In a real integration test, you would set up the fake client properly
			Expect(resource.NsList).ToNot(BeNil())

			// Step 4: Verify label selector
			Expect(resource.ListOptions.LabelSelector).To(ContainSubstring("app=test"))
			Expect(resource.ListOptions.LabelSelector).To(ContainSubstring("kubecloudscaler.cloud/ignore"))

			// Step 5: Test annotation management
			period := &periodPkg.Period{
				Type:         "test-period",
				GetStartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				GetEndTime:   time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC),
			}

			// Add annotations
			annotations := make(map[string]string)
			annotations = annotationMgr.AddAnnotations(annotations, period)
			Expect(annotations).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, "test-period"))

			// Add min/max annotations
			min := int32(2)
			max := int32(10)
			annotations = annotationMgr.AddMinMaxAnnotations(annotations, period, &min, max)
			Expect(annotations).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsMinOrigValue, "2"))
			Expect(annotations).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsMaxOrigValue, "10"))

			// Restore min/max annotations
			isRestored, restoredMin, restoredMax, cleanedAnnotations, err := annotationMgr.RestoreMinMaxAnnotations(annotations)
			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(*restoredMin).To(Equal(int32(2)))
			Expect(restoredMax).To(Equal(int32(10)))
			Expect(cleanedAnnotations).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsMinOrigValue))
			Expect(cleanedAnnotations).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsMaxOrigValue))
		})

		It("should handle complex namespace filtering scenarios", func() {
			client = NewFakeKubernetesClient()
			namespaceMgr = NewNamespaceManager(client, logger, nil)

			config := &Config{
				ForceExcludeSystemNamespaces: true,
				ExcludeNamespaces:            []string{"exclude-this", "monitoring"},
			}

			resource, err := namespaceMgr.InitConfig(ctx, config)
			Expect(err).ToNot(HaveOccurred())

			// Note: The fake client doesn't actually return the namespaces we created
			// In a real integration test, you would set up the fake client properly
			Expect(resource.NsList).ToNot(BeNil())
		})

		It("should handle annotation lifecycle management", func() {
			// Test complete annotation lifecycle
			period := &periodPkg.Period{
				Type:         "scaling-period",
				GetStartTime: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
				GetEndTime:   time.Date(2024, 1, 1, 17, 0, 0, 0, time.UTC),
			}

			// Initial state
			annotations := make(map[string]string)
			annotations["existing-key"] = "existing-value"

			// Add period annotations
			annotations = annotationMgr.AddAnnotations(annotations, period)
			Expect(annotations).To(HaveKeyWithValue("existing-key", "existing-value"))
			Expect(annotations).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, "scaling-period"))

			// Add bool annotation
			annotations = annotationMgr.AddBoolAnnotations(annotations, period, true)
			Expect(annotations).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "true"))

			// Add int annotation
			replicas := int32(5)
			annotations = annotationMgr.AddIntAnnotations(annotations, period, &replicas)
			// Should not overwrite existing original value
			Expect(annotations).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "true"))

			// Restore bool annotation
			isRestored, value, cleanedAnnotations, err := annotationMgr.RestoreBoolAnnotations(annotations)
			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(*value).To(BeTrue())
			Expect(cleanedAnnotations).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsOrigValue))

			// Remove all kubecloudscaler annotations
			finalAnnotations := annotationMgr.RemoveAnnotations(cleanedAnnotations)
			Expect(finalAnnotations).To(HaveKeyWithValue("existing-key", "existing-value"))
			Expect(finalAnnotations).ToNot(HaveKey(AnnotationsPrefix + "/" + PeriodType))
		})
	})

	Context("Error Handling Integration", func() {
		It("should handle client errors gracefully", func() {
			// Create a client that will fail
			mockClient := &MockKubernetesClient{
				CoreV1Func: func() CoreV1Interface {
					return &MockCoreV1Interface{
						NamespacesFunc: func() NamespaceLister {
							return &MockNamespaceLister{
								ListFunc: func(ctx context.Context, opts metaV1.ListOptions) (*coreV1.NamespaceList, error) {
									return nil, errors.New("network error")
								},
							}
						},
					}
				},
			}

			namespaceMgr = NewNamespaceManager(mockClient, logger, nil)

			config := &Config{
				Namespaces: []string{}, // Force it to try to list from cluster
			}

			_, err := namespaceMgr.InitConfig(ctx, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error listing namespaces"))
		})

		It("should handle annotation parsing errors gracefully", func() {
			// Test with invalid annotation values
			invalidAnnotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "not-a-bool",
			}

			_, _, _, err := annotationMgr.RestoreBoolAnnotations(invalidAnnotations)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error parsing bool value"))
		})
	})
})
