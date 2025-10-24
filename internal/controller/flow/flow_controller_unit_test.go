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

package controller

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

func TestFlowControllerUnit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Flow Controller Unit Tests")
}

var _ = Describe("FlowReconciler Unit Tests", func() {
	var (
		ctx        context.Context
		reconciler *FlowReconciler
		scheme     *runtime.Scheme
		fakeClient client.Client
		logger     *zerolog.Logger
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = &log.Logger
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&kubecloudscalerv1alpha3.Flow{}).
			Build()

		reconciler = &FlowReconciler{
			Client: fakeClient,
			Scheme: scheme,
			Logger: logger,
		}
	})

	Describe("extractFlowData", func() {
		It("should extract resource names and period names correctly", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "period1",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "resource1"},
								{Name: "resource2"},
							},
						},
						{
							PeriodName: "period2",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "resource2"},
								{Name: "resource3"},
							},
						},
					},
				},
			}

			resourceNames, periodNames, err := reconciler.extractFlowData(flow)

			Expect(err).NotTo(HaveOccurred())
			Expect(resourceNames).To(HaveKey("resource1"))
			Expect(resourceNames).To(HaveKey("resource2"))
			Expect(resourceNames).To(HaveKey("resource3"))
			Expect(periodNames).To(HaveKey("period1"))
			Expect(periodNames).To(HaveKey("period2"))
		})

		It("should handle empty flows", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Flows: []kubecloudscalerv1alpha3.Flows{},
				},
			}

			resourceNames, periodNames, err := reconciler.extractFlowData(flow)

			Expect(err).NotTo(HaveOccurred())
			Expect(resourceNames).To(BeEmpty())
			Expect(periodNames).To(BeEmpty())
		})
	})

	Describe("validatePeriodTimings", func() {
		It("should validate period timings successfully", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("test-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "test-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{
									Name:           "test-resource",
									StartTimeDelay: "1h",
									EndTimeDelay:   "1h",
								},
							},
						},
					},
				},
			}

			periodNames := map[string]bool{"test-period": true}
			err := reconciler.validatePeriodTimings(flow, periodNames)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail when period is not defined", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "non-existent-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "test-resource"},
							},
						},
					},
				},
			}

			periodNames := map[string]bool{"non-existent-period": true}
			err := reconciler.validatePeriodTimings(flow, periodNames)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("period non-existent-period referenced in flows but not defined"))
		})

		It("should fail when total delay exceeds period duration", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("short-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "10:00", // 1 hour period
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "short-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{
									Name:           "test-resource",
									StartTimeDelay: "30m",
									EndTimeDelay:   "45m", // Total: 1h 15m > 1h period
								},
							},
						},
					},
				},
			}

			periodNames := map[string]bool{"short-period": true}
			err := reconciler.validatePeriodTimings(flow, periodNames)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("total delay"))
			Expect(err.Error()).To(ContainSubstring("exceeds period duration"))
		})

		It("should fail with invalid delay format", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("test-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "test-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{
									Name:           "test-resource",
									StartTimeDelay: "invalid-delay",
								},
							},
						},
					},
				},
			}

			periodNames := map[string]bool{"test-period": true}
			err := reconciler.validatePeriodTimings(flow, periodNames)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid start time delay format"))
		})
	})

	Describe("calculatePeriodStartTime", func() {
		It("should calculate start time for recurring period", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						StartTime: "09:00",
					},
				},
			}

			startTime, err := reconciler.calculatePeriodStartTime(period, 0)

			Expect(err).NotTo(HaveOccurred())
			expectedTime, _ := time.Parse("15:04", "09:00")
			Expect(startTime.Format("15:04")).To(Equal(expectedTime.Format("15:04")))
		})

		It("should calculate start time for fixed period", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Fixed: &common.FixedPeriod{
						StartTime: "2024-01-01 09:00:00",
					},
				},
			}

			startTime, err := reconciler.calculatePeriodStartTime(period, 0)

			Expect(err).NotTo(HaveOccurred())
			expectedTime, _ := time.Parse("2006-01-02 15:04:05", "2024-01-01 09:00:00")
			Expect(startTime.Format("2006-01-02 15:04:05")).To(Equal(expectedTime.Format("2006-01-02 15:04:05")))
		})

		It("should fail with invalid time format", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						StartTime: "invalid-time",
					},
				},
			}

			_, err := reconciler.calculatePeriodStartTime(period, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse recurring start time"))
		})

		It("should fail when no valid time period found", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{},
			}

			_, err := reconciler.calculatePeriodStartTime(period, 0)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no valid time period found"))
		})
	})

	Describe("calculatePeriodEndTime", func() {
		It("should calculate end time for recurring period", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						EndTime: "17:00",
					},
				},
			}

			endTime, err := reconciler.calculatePeriodEndTime(period, 0)

			Expect(err).NotTo(HaveOccurred())
			expectedTime, _ := time.Parse("15:04", "17:00")
			Expect(endTime.Format("15:04")).To(Equal(expectedTime.Format("15:04")))
		})

		It("should calculate end time for fixed period", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Fixed: &common.FixedPeriod{
						EndTime: "2024-01-01 17:00:00",
					},
				},
			}

			endTime, err := reconciler.calculatePeriodEndTime(period, 0)

			Expect(err).NotTo(HaveOccurred())
			expectedTime, _ := time.Parse("2006-01-02 15:04:05", "2024-01-01 17:00:00")
			Expect(endTime.Format("2006-01-02 15:04:05")).To(Equal(expectedTime.Format("2006-01-02 15:04:05")))
		})
	})

	Describe("getPeriodDuration", func() {
		It("should calculate duration for recurring period", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						StartTime: "09:00",
						EndTime:   "17:00",
					},
				},
			}

			duration, err := reconciler.getPeriodDuration(period)

			Expect(err).NotTo(HaveOccurred())
			Expect(duration).To(Equal(8 * time.Hour))
		})

		It("should calculate duration for fixed period", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Fixed: &common.FixedPeriod{
						StartTime: "2024-01-01 09:00:00",
						EndTime:   "2024-01-01 17:00:00",
					},
				},
			}

			duration, err := reconciler.getPeriodDuration(period)

			Expect(err).NotTo(HaveOccurred())
			Expect(duration).To(Equal(8 * time.Hour))
		})

		It("should fail when end time is before start time", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						StartTime: "17:00",
						EndTime:   "09:00",
					},
				},
			}

			_, err := reconciler.getPeriodDuration(period)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("end time is before start time"))
		})
	})

	Describe("createResourceMappings", func() {
		It("should create resource mappings for K8s resources", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("test-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Resources: kubecloudscalerv1alpha3.Resources{
						K8s: []kubecloudscalerv1alpha3.K8sResource{
							{
								Name: "test-k8s",
								Resources: common.Resources{
									Types: []string{"deployments"},
									Names: []string{"test-deployment"},
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "test-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "test-k8s"},
							},
						},
					},
				},
			}

			resourceNames := map[string]bool{"test-k8s": true}
			mappings, err := reconciler.createResourceMappings(flow, resourceNames)

			Expect(err).NotTo(HaveOccurred())
			Expect(mappings).To(HaveKey("test-k8s"))
			Expect(mappings["test-k8s"].Type).To(Equal("k8s"))
			Expect(mappings["test-k8s"].Periods).To(HaveLen(1))
		})

		It("should create resource mappings for GCP resources", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("test-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Resources: kubecloudscalerv1alpha3.Resources{
						Gcp: []kubecloudscalerv1alpha3.GcpResource{
							{
								Name: "test-gcp",
								Resources: common.Resources{
									Types: []string{"vm-instances"},
									Names: []string{"test-instance"},
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "test-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{Name: "test-gcp"},
							},
						},
					},
				},
			}

			resourceNames := map[string]bool{"test-gcp": true}
			mappings, err := reconciler.createResourceMappings(flow, resourceNames)

			Expect(err).NotTo(HaveOccurred())
			Expect(mappings).To(HaveKey("test-gcp"))
			Expect(mappings["test-gcp"].Type).To(Equal("gcp"))
			Expect(mappings["test-gcp"].Periods).To(HaveLen(1))
		})

		It("should fail when resource is defined in both K8s and GCP", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Resources: kubecloudscalerv1alpha3.Resources{
						K8s: []kubecloudscalerv1alpha3.K8sResource{
							{Name: "duplicate-resource"},
						},
						Gcp: []kubecloudscalerv1alpha3.GcpResource{
							{Name: "duplicate-resource"},
						},
					},
				},
			}

			resourceNames := map[string]bool{"duplicate-resource": true}
			_, err := reconciler.createResourceMappings(flow, resourceNames)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource duplicate-resource is defined in both K8s and GCP resources"))
		})

		It("should fail when resource is not defined", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Resources: kubecloudscalerv1alpha3.Resources{},
				},
			}

			resourceNames := map[string]bool{"undefined-resource": true}
			_, err := reconciler.createResourceMappings(flow, resourceNames)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("resource undefined-resource referenced in flows but not defined in resources"))
		})
	})

	Describe("updateFlowStatus", func() {
		It("should update flow status successfully", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow",
					Namespace: "default",
				},
				Spec: kubecloudscalerv1alpha3.FlowSpec{},
			}

			Expect(fakeClient.Create(ctx, flow)).To(Succeed())

			condition := metav1.Condition{
				Type:    "Processed",
				Status:  metav1.ConditionTrue,
				Reason:  "ProcessingSucceeded",
				Message: "Flow processed successfully",
			}

			result, err := reconciler.updateFlowStatus(ctx, flow, condition)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			// Verify the condition was added
			updatedFlow := &kubecloudscalerv1alpha3.Flow{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(flow), updatedFlow)).To(Succeed())
			Expect(updatedFlow.Status.Conditions).To(HaveLen(1))
			Expect(updatedFlow.Status.Conditions[0].Type).To(Equal("Processed"))
		})

		It("should update existing condition", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow-update",
					Namespace: "default",
				},
				Spec: kubecloudscalerv1alpha3.FlowSpec{},
				Status: kubecloudscalerv1alpha3.FlowStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "Processed",
							Status: metav1.ConditionFalse,
							Reason: "ProcessingFailed",
						},
					},
				},
			}

			Expect(fakeClient.Create(ctx, flow)).To(Succeed())

			condition := metav1.Condition{
				Type:    "Processed",
				Status:  metav1.ConditionTrue,
				Reason:  "ProcessingSucceeded",
				Message: "Flow processed successfully",
			}

			_, err := reconciler.updateFlowStatus(ctx, flow, condition)

			Expect(err).NotTo(HaveOccurred())

			// Verify the condition was updated
			updatedFlow := &kubecloudscalerv1alpha3.Flow{}
			Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(flow), updatedFlow)).To(Succeed())
			Expect(updatedFlow.Status.Conditions).To(HaveLen(1))
			Expect(updatedFlow.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
		})
	})
})
