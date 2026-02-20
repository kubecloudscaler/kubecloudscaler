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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

func TestAnnotationManagerOriginal(t *testing.T) {
	RegisterFailHandler(Fail)
	// RunSpecs is handled by suite_test.go
}

var _ = Describe("AnnotationManager", func() {
	var (
		annotationMgr AnnotationManager
		mockPeriod    *MockPeriod
	)

	BeforeEach(func() {
		annotationMgr = NewAnnotationManager()
		mockPeriod = &MockPeriod{
			Type:      "test-period",
			StartTime: &mockTime{timeStr: "2024-01-01T00:00:00Z"},
			EndTime:   &mockTime{timeStr: "2024-01-01T01:00:00Z"},
			Timezone:  ptr.To("UTC"),
		}
	})

	Context("AddAnnotations", func() {
		It("should add period annotations to empty map", func() {
			annotations := make(map[string]string)

			result := annotationMgr.AddAnnotations(annotations, mockPeriod)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, "test-period"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodStartTime, "2024-01-01T00:00:00Z"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodEndTime, "2024-01-01T01:00:00Z"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodTimezone, "UTC"))
		})

		It("should add period annotations to existing map", func() {
			annotations := map[string]string{
				"existing-key": "existing-value",
			}

			result := annotationMgr.AddAnnotations(annotations, mockPeriod)

			Expect(result).To(HaveKeyWithValue("existing-key", "existing-value"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, "test-period"))
		})

		It("should handle nil annotations map", func() {
			result := annotationMgr.AddAnnotations(nil, mockPeriod)

			Expect(result).ToNot(BeNil())
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, "test-period"))
		})

		It("should handle invalid period type", func() {
			annotations := make(map[string]string)

			result := annotationMgr.AddAnnotations(annotations, "invalid-period")

			Expect(result).To(Equal(annotations))
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

			result := annotationMgr.RemoveAnnotations(annotations)

			Expect(result).To(HaveKeyWithValue("other-annotation", "value"))
			Expect(result).ToNot(HaveKey("kubecloudscaler.cloud/period-type"))
			Expect(result).ToNot(HaveKey("kubecloudscaler.cloud/period-start"))
			Expect(result).ToNot(HaveKey("kubecloudscaler.cloud/ignore"))
		})

		It("should handle empty annotations map", func() {
			annotations := map[string]string{}

			result := annotationMgr.RemoveAnnotations(annotations)

			Expect(result).To(BeEmpty())
		})

		It("should handle nil annotations map", func() {
			result := annotationMgr.RemoveAnnotations(nil)

			Expect(result).To(BeNil())
		})
	})

	Context("AddMinMaxAnnotations", func() {
		It("should add min/max annotations with original values", func() {
			annotations := make(map[string]string)
			min := int32(2)
			max := int32(10)

			result := annotationMgr.AddMinMaxAnnotations(annotations, mockPeriod, &min, max)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsMinOrigValue, "2"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsMaxOrigValue, "10"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, "test-period"))
		})

		It("should not overwrite existing original values", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsMinOrigValue: "existing",
				AnnotationsPrefix + "/" + AnnotationsMaxOrigValue: "existing",
			}
			min := int32(2)
			max := int32(10)

			result := annotationMgr.AddMinMaxAnnotations(annotations, mockPeriod, &min, max)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsMinOrigValue, "existing"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsMaxOrigValue, "existing"))
		})

		It("should handle nil min value", func() {
			annotations := make(map[string]string)
			max := int32(10)

			result := annotationMgr.AddMinMaxAnnotations(annotations, mockPeriod, nil, max)

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

			isRestored, min, max, result, err := annotationMgr.RestoreMinMaxAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(*min).To(Equal(int32(5)))
			Expect(max).To(Equal(int32(20)))
			Expect(result).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsMinOrigValue))
			Expect(result).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsMaxOrigValue))
		})

		It("should return true when no annotations exist", func() {
			annotations := map[string]string{}

			isRestored, min, max, result, err := annotationMgr.RestoreMinMaxAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeTrue())
			Expect(*min).To(Equal(int32(0)))
			Expect(max).To(Equal(int32(0)))
			Expect(result).To(BeEmpty())
		})

		It("should return error for invalid min value", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsMinOrigValue: "invalid",
			}

			isRestored, min, max, _, err := annotationMgr.RestoreMinMaxAnnotations(annotations)

			Expect(err).To(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(min).To(BeNil())
			Expect(max).To(Equal(int32(0)))
		})

		It("should return error for invalid max value", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsMaxOrigValue: "invalid",
			}

			isRestored, min, max, _, err := annotationMgr.RestoreMinMaxAnnotations(annotations)

			Expect(err).To(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(min).To(BeNil())
			Expect(max).To(Equal(int32(0)))
		})
	})

	Context("AddBoolAnnotations", func() {
		It("should add bool annotation with original value", func() {
			annotations := make(map[string]string)
			value := true

			result := annotationMgr.AddBoolAnnotations(annotations, mockPeriod, value)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "true"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, "test-period"))
		})

		It("should not overwrite existing original value", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "false",
			}

			result := annotationMgr.AddBoolAnnotations(annotations, mockPeriod, true)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "false"))
		})
	})

	Context("RestoreBoolAnnotations", func() {
		It("should restore bool value from annotations", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "true",
			}

			isRestored, value, result, err := annotationMgr.RestoreBoolAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(*value).To(BeTrue())
			Expect(result).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsOrigValue))
		})

		It("should return true when no annotation exists", func() {
			annotations := map[string]string{}

			isRestored, value, result, err := annotationMgr.RestoreBoolAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeTrue())
			Expect(*value).To(BeFalse())
			Expect(result).To(BeEmpty())
		})

		It("should return error for invalid bool value", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "invalid",
			}

			isRestored, value, _, err := annotationMgr.RestoreBoolAnnotations(annotations)

			Expect(err).To(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(value).To(BeNil())
		})
	})

	Context("AddIntAnnotations", func() {
		It("should add int annotation with original value", func() {
			annotations := make(map[string]string)
			value := int32(42)

			result := annotationMgr.AddIntAnnotations(annotations, mockPeriod, &value)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "42"))
			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+PeriodType, "test-period"))
		})

		It("should handle nil int value", func() {
			annotations := make(map[string]string)

			result := annotationMgr.AddIntAnnotations(annotations, mockPeriod, nil)

			Expect(result).To(HaveKeyWithValue(AnnotationsPrefix+"/"+AnnotationsOrigValue, "0"))
		})
	})

	Context("RestoreIntAnnotations", func() {
		It("should restore int value from annotations", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "42",
			}

			isRestored, value, result, err := annotationMgr.RestoreIntAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeFalse())
			Expect(*value).To(Equal(int32(42)))
			Expect(result).ToNot(HaveKey(AnnotationsPrefix + "/" + AnnotationsOrigValue))
		})

		It("should return true when no annotation exists", func() {
			annotations := map[string]string{}

			isRestored, value, result, err := annotationMgr.RestoreIntAnnotations(annotations)

			Expect(err).ToNot(HaveOccurred())
			Expect(isRestored).To(BeTrue())
			Expect(*value).To(Equal(int32(0)))
			Expect(result).To(BeEmpty())
		})

		It("should return error for invalid int value", func() {
			annotations := map[string]string{
				AnnotationsPrefix + "/" + AnnotationsOrigValue: "invalid",
			}

			isRestored, value, _, err := annotationMgr.RestoreIntAnnotations(annotations)

			Expect(err).To(HaveOccurred())
			// The annotation exists but is corrupted: restoration has NOT happened, so
			// isRestored must be false (the old code incorrectly returned true here).
			Expect(isRestored).To(BeFalse())
			Expect(value).To(BeNil())
		})
	})
})

// Mock types for testing
type MockPeriod struct {
	Type      string
	StartTime interface{ String() string }
	EndTime   interface{ String() string }
	Timezone  *string
}

func (m *MockPeriod) GetType() string {
	return m.Type
}

func (m *MockPeriod) GetStartTime() interface{ String() string } {
	return m.StartTime
}

func (m *MockPeriod) GetEndTime() interface{ String() string } {
	return m.EndTime
}

func (m *MockPeriod) GetTimezone() *string {
	return m.Timezone
}

type mockTime struct {
	timeStr string
}

func (m *mockTime) String() string {
	return m.timeStr
}
