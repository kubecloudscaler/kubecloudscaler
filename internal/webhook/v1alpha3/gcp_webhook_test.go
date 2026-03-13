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

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

var _ = Describe("Gcp Webhook Validation", func() {
	var (
		validator *GcpCustomValidator
		ctx       context.Context
	)

	BeforeEach(func() {
		validator = &GcpCustomValidator{}
		ctx = context.Background()
	})

	Context("When validating Gcp creation", func() {
		It("should accept valid minimal spec", func() {
			gcp := &kubecloudscalerv1alpha3.Gcp{
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID: "my-project",
					},
					Periods: []common.ScalerPeriod{
						{
							Type: common.PeriodTypeUp,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayMonday, common.DayTuesday},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
					Resources: common.Resources{
						Types: []common.ResourceKind{common.ResourceVMInstances},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, gcp)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})

		It("should reject empty projectId", func() {
			gcp := &kubecloudscalerv1alpha3.Gcp{
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID: "",
					},
					Periods: []common.ScalerPeriod{
						{
							Type: common.PeriodTypeUp,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayMonday},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, gcp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("config.projectId is required"))
			Expect(warnings).To(BeNil())
		})

		It("should reject empty periods", func() {
			gcp := &kubecloudscalerv1alpha3.Gcp{
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID: "my-project",
					},
					Periods: []common.ScalerPeriod{},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, gcp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one period is required"))
			Expect(warnings).To(BeNil())
		})

		It("should reject invalid period type", func() {
			gcp := &kubecloudscalerv1alpha3.Gcp{
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID: "my-project",
					},
					Periods: []common.ScalerPeriod{
						{
							Type: "invalid",
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayMonday},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, gcp)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type must be 'up' or 'down'"))
			Expect(warnings).To(BeNil())
		})
	})

	Context("When validating Gcp updates", func() {
		It("should accept valid spec on update", func() {
			oldGcp := &kubecloudscalerv1alpha3.Gcp{
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID: "my-project",
					},
					Periods: []common.ScalerPeriod{
						{
							Type: common.PeriodTypeUp,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayMonday},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
				},
			}

			newGcp := &kubecloudscalerv1alpha3.Gcp{
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID: "my-project",
					},
					Periods: []common.ScalerPeriod{
						{
							Type: common.PeriodTypeDown,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayMonday, common.DayFriday},
									StartTime: "18:00",
									EndTime:   "08:00",
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateUpdate(ctx, oldGcp, newGcp)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})

	Context("When validating Gcp deletion", func() {
		It("should always allow deletion", func() {
			gcp := &kubecloudscalerv1alpha3.Gcp{
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID: "my-project",
					},
					Periods: []common.ScalerPeriod{
						{
							Type: common.PeriodTypeUp,
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayMonday},
									StartTime: "09:00",
									EndTime:   "17:00",
								},
							},
						},
					},
				},
			}

			warnings, err := validator.ValidateDelete(ctx, gcp)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})
})
