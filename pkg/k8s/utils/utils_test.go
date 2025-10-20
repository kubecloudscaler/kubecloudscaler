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
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

func TestUtilsOriginal(t *testing.T) {
	RegisterFailHandler(Fail)
	// RunSpecs is handled by suite_test.go
}

// BeforeSuite is handled by suite_test.go

var _ = Describe("Utils", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("SetNamespaceList", func() {
		var (
			config *Config
			client kubernetes.Interface
		)

		BeforeEach(func() {
			client = fake.NewSimpleClientset()
			config = &Config{
				Client:                       client,
				ForceExcludeSystemNamespaces: true,
			}
		})

		It("should return specified namespaces when provided", func() {
			config.Namespaces = []string{"namespace1", "namespace2", "namespace3"}

			nsList, err := SetNamespaceList(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(Equal([]string{"namespace1", "namespace2", "namespace3"}))
		})

		It("should return all namespaces from cluster when no namespaces specified", func() {
			// Create test namespaces
			ns1 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-1"}}
			ns2 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-2"}}
			ns3 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "kube-system"}}

			client = fake.NewSimpleClientset(ns1, ns2, ns3)
			config.Client = client

			nsList, err := SetNamespaceList(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElements("test-ns-1", "test-ns-2"))
			Expect(nsList).ToNot(ContainElement("kube-system"))
		})

		It("should exclude specified namespaces", func() {
			ns1 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-1"}}
			ns2 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "exclude-ns"}}
			ns3 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-2"}}

			client = fake.NewSimpleClientset(ns1, ns2, ns3)
			config.Client = client
			config.ExcludeNamespaces = []string{"exclude-ns"}

			nsList, err := SetNamespaceList(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElements("test-ns-1", "test-ns-2"))
			Expect(nsList).ToNot(ContainElement("exclude-ns"))
		})

		It("should force exclude system namespaces when enabled", func() {
			ns1 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-1"}}
			ns2 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "kube-system"}}
			ns3 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "kube-public"}}

			client = fake.NewSimpleClientset(ns1, ns2, ns3)
			config.Client = client

			nsList, err := SetNamespaceList(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElement("test-ns-1"))
			Expect(nsList).ToNot(ContainElements("kube-system", "kube-public"))
		})

		It("should always exclude own namespace", func() {
			ns1 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns-1"}}
			ns2 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "my-namespace"}}

			client = fake.NewSimpleClientset(ns1, ns2)
			config.Client = client

			// Set POD_NAMESPACE environment variable
			os.Setenv("POD_NAMESPACE", "my-namespace")
			defer os.Unsetenv("POD_NAMESPACE")

			nsList, err := SetNamespaceList(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElement("test-ns-1"))
			Expect(nsList).ToNot(ContainElement("my-namespace"))
		})

		It("should return empty list when no namespaces are found", func() {
			// Create a fake client with no namespaces
			config.Client = fake.NewSimpleClientset()
			config.Namespaces = []string{} // Force it to try to list from cluster

			nsList, err := SetNamespaceList(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(BeEmpty())
		})
	})

	Context("addAnnotations", func() {
		It("should add period annotations to empty map", func() {
			period := createTestPeriod()
			annotations := make(map[string]string)

			result := addAnnotations(annotations, period)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, period.Type))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodStartTime, period.GetStartTime.String()))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodEndTime, period.GetEndTime.String()))
		})

		It("should add period annotations to existing map", func() {
			period := createTestPeriod()
			annotations := map[string]string{
				"existing-key": "existing-value",
			}

			result := addAnnotations(annotations, period)

			Expect(result).To(HaveKeyWithValue("existing-key", "existing-value"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, period.Type))
		})

		It("should handle nil annotations map", func() {
			period := createTestPeriod()

			result := addAnnotations(nil, period)

			Expect(result).ToNot(BeNil())
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, period.Type))
		})
	})

	Context("RemoveAnnotations", func() {
		It("should remove all kubecloudscaler annotations", func() {
			annotations := map[string]string{
				"kubecloudscaler.cloud/period-type":  "test",
				"kubecloudscaler.cloud/period-start": "2024-01-01T00:00:00Z",
				"other-annotation":                   "value",
				"kubecloudscaler.cloud/ignore":       "true",
			}

			result := RemoveAnnotations(annotations)

			Expect(result).To(HaveKeyWithValue("other-annotation", "value"))
			Expect(result).ToNot(HaveKey("kubecloudscaler.cloud/period-type"))
			Expect(result).ToNot(HaveKey("kubecloudscaler.cloud/period-start"))
			Expect(result).ToNot(HaveKey("kubecloudscaler.cloud/ignore"))
		})

		It("should handle empty annotations map", func() {
			annotations := map[string]string{}

			result := RemoveAnnotations(annotations)

			Expect(result).To(BeEmpty())
		})

		It("should handle nil annotations map", func() {
			result := RemoveAnnotations(nil)

			Expect(result).To(BeNil())
		})
	})

	Context("PrepareSearch", func() {
		var (
			config *Config
			client kubernetes.Interface
		)

		BeforeEach(func() {
			client = fake.NewSimpleClientset()
			config = &Config{
				Client: client,
			}
		})

		It("should prepare search with default label selector", func() {
			ns1 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns"}}
			client = fake.NewSimpleClientset(ns1)
			config.Client = client

			nsList, listOptions, err := PrepareSearch(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElement("test-ns"))
			Expect(listOptions.LabelSelector).To(ContainSubstring("kubecloudscaler.cloud/ignore"))
		})

		It("should merge custom label selector with default", func() {
			ns1 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns"}}
			client = fake.NewSimpleClientset(ns1)
			config.Client = client
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

			nsList, listOptions, err := PrepareSearch(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElement("test-ns"))
			Expect(listOptions.LabelSelector).To(ContainSubstring("app=test"))
			Expect(listOptions.LabelSelector).To(ContainSubstring("environment in (prod,staging)"))
			Expect(listOptions.LabelSelector).To(ContainSubstring("kubecloudscaler.cloud/ignore"))
		})

		It("should skip ignore label in custom expressions", func() {
			ns1 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns"}}
			client = fake.NewSimpleClientset(ns1)
			config.Client = client
			config.LabelSelector = &metaV1.LabelSelector{
				MatchExpressions: []metaV1.LabelSelectorRequirement{
					{
						Key:      AnnotationsPrefix + "/ignore",
						Operator: metaV1.LabelSelectorOpExists,
						Values:   []string{},
					},
				},
			}

			nsList, listOptions, err := PrepareSearch(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(ContainElement("test-ns"))
			// Should only have the default ignore expression, not the custom one
			Expect(listOptions.LabelSelector).To(ContainSubstring("kubecloudscaler.cloud/ignore"))
		})

		It("should work with empty namespace list", func() {
			config.Client = fake.NewSimpleClientset()
			config.Namespaces = []string{} // Force it to try to list from cluster

			nsList, listOptions, err := PrepareSearch(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(nsList).To(BeEmpty())
			Expect(listOptions.LabelSelector).ToNot(BeEmpty())
		})
	})

	Context("InitConfig", func() {
		var (
			config *Config
			client kubernetes.Interface
		)

		BeforeEach(func() {
			client = fake.NewSimpleClientset()
			config = &Config{
				Client: client,
				Period: createTestPeriod(),
			}
		})

		It("should initialize K8sResource successfully", func() {
			ns1 := &coreV1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: "test-ns"}}
			client = fake.NewSimpleClientset(ns1)
			config.Client = client

			resource, err := InitConfig(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(resource).ToNot(BeNil())
			Expect(resource.Period).To(Equal(config.Period))
			Expect(resource.NsList).To(ContainElement("test-ns"))
			Expect(resource.ListOptions.LabelSelector).ToNot(BeEmpty())
		})

		It("should work with empty namespace list", func() {
			config.Client = fake.NewSimpleClientset()
			config.Namespaces = []string{} // Force it to try to list from cluster

			resource, err := InitConfig(ctx, config)

			Expect(err).ToNot(HaveOccurred())
			Expect(resource).ToNot(BeNil())
			Expect(resource.NsList).To(BeEmpty())
			Expect(resource.ListOptions.LabelSelector).ToNot(BeEmpty())
		})
	})

	Context("AddMinMaxAnnotations", func() {
		It("should add min/max annotations with original values", func() {
			period := createTestPeriod()
			annotations := make(map[string]string)
			min := int32(2)
			max := int32(10)

			result := AddMinMaxAnnotations(annotations, period, &min, max)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsMinOrigValue, "2"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsMaxOrigValue, "10"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, period.Type))
		})

		It("should not overwrite existing original values", func() {
			period := createTestPeriod()
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "existing",
			}
			min := int32(2)
			max := int32(10)

			result := AddMinMaxAnnotations(annotations, period, &min, max)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "existing"))
			Expect(result).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsMinOrigValue))
			Expect(result).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsMaxOrigValue))
		})

		It("should handle nil min value", func() {
			period := createTestPeriod()
			annotations := make(map[string]string)
			max := int32(10)

			result := AddMinMaxAnnotations(annotations, period, nil, max)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsMinOrigValue, "0"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsMaxOrigValue, "10"))
		})
	})

	Context("RestoreMinMaxAnnotations", func() {
		It("should restore min/max values from annotations", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsMinOrigValue: "5",
				AnnotationsPrefix + "/" + AnnotationsMaxOrigValue: "20",
			}

			isRestored, min, max, result, err := RestoreMinMaxAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(*min).To(Equal(int32(5)))
			Expect(max).To(Equal(int32(20)))
			Expect(result).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsMinOrigValue))
			Expect(result).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsMaxOrigValue))
		})

		It("should return true when no annotations exist", func() {
			annotations := map[string]string{}

			isRestored, min, max, result, err := RestoreMinMaxAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeTrue())
			Expect(min).ToNot(BeNil())
			Expect(*min).To(Equal(int32(0)))
			Expect(max).To(Equal(int32(0)))
			Expect(result).To(BeEmpty())
		})

		It("should return error for invalid min value", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsMinOrigValue: "invalid",
			}

			isRestored, min, max, _, err := RestoreMinMaxAnnotations(annotations)

			Expect(err).To(HaveOccurred())
			Expect(isRestored).To(BeTrue())
			Expect(min).To(BeNil())
			Expect(max).To(Equal(int32(0)))
		})

		It("should return error for invalid max value", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsMaxOrigValue: "invalid",
			}

			isRestored, min, max, _, err := RestoreMinMaxAnnotations(annotations)

			Expect(err).To(HaveOccurred())
			Expect(isRestored).To(BeTrue())
			Expect(min).To(BeNil())
			Expect(max).To(Equal(int32(0)))
		})
	})

	Context("AddBoolAnnotations", func() {
		It("should add bool annotation with original value", func() {
			period := createTestPeriod()
			annotations := make(map[string]string)
			value := true

			result := AddBoolAnnotations(annotations, period, value)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "true"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, period.Type))
		})

		It("should not overwrite existing original value", func() {
			period := createTestPeriod()
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "false",
			}

			result := AddBoolAnnotations(annotations, period, true)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "false"))
		})
	})

	Context("RestoreBoolAnnotations", func() {
		It("should restore bool value from annotations", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "true",
			}

			isRestored, value, result, err := RestoreBoolAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(*value).To(BeTrue())
			Expect(result).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsOrigValue))
		})

		It("should return true when no annotation exists", func() {
			annotations := map[string]string{}

			isRestored, value, result, err := RestoreBoolAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeTrue())
			Expect(value).ToNot(BeNil())
			Expect(*value).To(BeFalse())
			Expect(result).To(BeEmpty())
		})

		It("should return error for invalid bool value", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "invalid",
			}

			isRestored, value, _, err := RestoreBoolAnnotations(annotations)

			Expect(err).To(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(value).To(BeNil())
		})
	})

	Context("AddIntAnnotations", func() {
		It("should add int annotation with original value", func() {
			period := createTestPeriod()
			annotations := make(map[string]string)
			value := int32(42)

			result := AddIntAnnotations(annotations, period, &value)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "42"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, period.Type))
		})

		It("should handle nil int value", func() {
			period := createTestPeriod()
			annotations := make(map[string]string)

			result := AddIntAnnotations(annotations, period, nil)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "0"))
		})
	})

	Context("RestoreIntAnnotations", func() {
		It("should restore int value from annotations", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "42",
			}

			isRestored, value, result, err := RestoreIntAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(*value).To(Equal(int32(42)))
			Expect(result).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsOrigValue))
		})

		It("should return true when no annotation exists", func() {
			annotations := map[string]string{}

			isRestored, value, result, err := RestoreIntAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeTrue())
			Expect(value).ToNot(BeNil())
			Expect(*value).To(Equal(int32(0)))
			Expect(result).To(BeEmpty())
		})

		It("should return error for invalid int value", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "invalid",
			}

			isRestored, value, _, err := RestoreIntAnnotations(annotations)

			Expect(err).To(HaveOccurred())
			Expect(isRestored).To(BeTrue())
			Expect(value).To(BeNil())
		})
	})
})

// Helper functions and mocks

func createTestPeriod() *periodPkg.Period {
	now := time.Now()
	return &periodPkg.Period{
		Type:         "test-period",
		GetStartTime: now,
		GetEndTime:   now.Add(time.Hour),
		Period: &common.RecurringPeriod{
			Timezone: ptr.To("UTC"),
		},
	}
}
