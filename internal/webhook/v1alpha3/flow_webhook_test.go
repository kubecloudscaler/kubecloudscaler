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

package v1alpha3

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

var _ = Describe("Flow Webhook Validation", func() {
	var (
		validator *FlowCustomValidator
		ctx       context.Context
	)

	BeforeEach(func() {
		validator = &FlowCustomValidator{}
		ctx = context.Background()
	})

	Context("When validating Flow creation", func() {
		It("should accept valid flow", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("test-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday", "Tuesday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Resources: kubecloudscalerv1alpha3.Resources{
						K8s: []kubecloudscalerv1alpha3.K8sResource{
							{
								Name: "test-k8s-resource",
								Resources: common.Resources{
									Types: []string{"deployments"},
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "test-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{
									Name:  "test-k8s-resource",
									Delay: ptr.To("600s"),
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, flow)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})

		It("should reject flow with duplicate period names", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("duplicate-name"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
						{
							Type: "down",
							Name: ptr.To("duplicate-name"), // Duplicate name
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Tuesday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, flow)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate period name"))
			Expect(warnings).To(BeNil())
		})

		It("should reject flow with missing period names", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
						{
							Type: "up",
							// Name is nil
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, flow)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("has no name"))
			Expect(warnings).To(BeNil())
		})

		It("should reject flow with duplicate K8s resource names", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
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
								Name: "duplicate-name",
								Resources: common.Resources{
									Types: []string{"deployments"},
								},
							},
							{
								Name: "duplicate-name", // Duplicate name
								Resources: common.Resources{
									Types: []string{"statefulsets"},
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, flow)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate K8s resource name"))
			Expect(warnings).To(BeNil())
		})

		It("should reject flow with cross-type resource name conflicts", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
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
								Name: "conflict-name",
								Resources: common.Resources{
									Types: []string{"deployments"},
								},
							},
						},
						Gcp: []kubecloudscalerv1alpha3.GcpResource{
							{
								Name: "conflict-name", // Same name as K8s resource
								Resources: common.Resources{
									Types: []string{"vm-instances"},
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, flow)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is used in both K8s and GCP resources"))
			Expect(warnings).To(BeNil())
		})

		It("should reject flow with timing constraints violation", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
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
					Resources: kubecloudscalerv1alpha3.Resources{
						K8s: []kubecloudscalerv1alpha3.K8sResource{
							{
								Name: "test-k8s-resource",
								Resources: common.Resources{
									Types: []string{"deployments"},
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "short-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{
									Name:  "test-k8s-resource",
									Delay: ptr.To("7200s"), // Exceeds 1 hour period
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, flow)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds period duration"))
			Expect(warnings).To(BeNil())
		})

		It("should reject flow with invalid delay format", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
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
								Name: "test-k8s-resource",
								Resources: common.Resources{
									Types: []string{"deployments"},
								},
							},
						},
					},
					Flows: []kubecloudscalerv1alpha3.Flows{
						{
							PeriodName: "test-period",
							Resources: []kubecloudscalerv1alpha3.FlowResource{
								{
									Name:  "test-k8s-resource",
									Delay: ptr.To("invalid-duration"), // Invalid format
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, flow)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid delay format"))
			Expect(warnings).To(BeNil())
		})
	})

	Context("When validating Flow updates", func() {
		It("should accept valid flow update", func() {
			oldFlow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
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
				},
			}

			newFlow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("test-period"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday", "Tuesday"}, // Added Tuesday
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateUpdate(ctx, oldFlow, newFlow)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})

		It("should reject invalid flow update", func() {
			oldFlow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
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
				},
			}

			newFlow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
						{
							Type: "up",
							Name: ptr.To("duplicate-name"),
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Monday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
						{
							Type: "down",
							Name: ptr.To("duplicate-name"), // Duplicate name
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []string{"Tuesday"},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateUpdate(ctx, oldFlow, newFlow)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate period name"))
			Expect(warnings).To(BeNil())
		})
	})

	Context("When validating Flow deletion", func() {
		It("should always allow deletion", func() {
			flow := &kubecloudscalerv1alpha3.Flow{
				Spec: kubecloudscalerv1alpha3.FlowSpec{
					Periods: []*common.ScalerPeriod{
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
				},
			}

			warnings, err := validator.ValidateDelete(ctx, flow)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})

	Context("When validating invalid object types", func() {
		It("should reject non-Flow objects in ValidateCreate", func() {
			// Create a mock object that implements runtime.Object but is not a Flow
			invalidObj := &kubecloudscalerv1alpha3.K8s{}

			warnings, err := validator.ValidateCreate(ctx, invalidObj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a Flow object"))
			Expect(warnings).To(BeNil())
		})

		It("should reject non-Flow objects in ValidateUpdate", func() {
			// Create mock objects that implement runtime.Object but are not Flow
			oldObj := &kubecloudscalerv1alpha3.K8s{}
			newObj := &kubecloudscalerv1alpha3.K8s{}

			warnings, err := validator.ValidateUpdate(ctx, oldObj, newObj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a Flow object"))
			Expect(warnings).To(BeNil())
		})

		It("should reject non-Flow objects in ValidateDelete", func() {
			// Create a mock object that implements runtime.Object but is not a Flow
			invalidObj := &kubecloudscalerv1alpha3.K8s{}

			warnings, err := validator.ValidateDelete(ctx, invalidObj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a Flow object"))
			Expect(warnings).To(BeNil())
		})
	})
})
