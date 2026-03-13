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

var _ = Describe("K8s Webhook Validation", func() {
	var (
		validator *K8sCustomValidator
		ctx       context.Context
	)

	BeforeEach(func() {
		validator = &K8sCustomValidator{}
		ctx = context.Background()
	})

	Context("When validating K8s creation", func() {
		It("should accept valid minimal spec", func() {
			k8s := &kubecloudscalerv1alpha3.K8s{
				Spec: kubecloudscalerv1alpha3.K8sSpec{
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
						Types: []common.ResourceKind{common.ResourceDeployments},
					},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, k8s)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})

		It("should reject empty periods", func() {
			k8s := &kubecloudscalerv1alpha3.K8s{
				Spec: kubecloudscalerv1alpha3.K8sSpec{
					Periods: []common.ScalerPeriod{},
				},
			}

			warnings, err := validator.ValidateCreate(ctx, k8s)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one period is required"))
			Expect(warnings).To(BeNil())
		})

		It("should reject invalid period type", func() {
			k8s := &kubecloudscalerv1alpha3.K8s{
				Spec: kubecloudscalerv1alpha3.K8sSpec{
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

			warnings, err := validator.ValidateCreate(ctx, k8s)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type must be 'up' or 'down'"))
			Expect(warnings).To(BeNil())
		})
	})

	Context("When validating K8s updates", func() {
		It("should accept valid spec on update", func() {
			oldK8s := &kubecloudscalerv1alpha3.K8s{
				Spec: kubecloudscalerv1alpha3.K8sSpec{
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

			newK8s := &kubecloudscalerv1alpha3.K8s{
				Spec: kubecloudscalerv1alpha3.K8sSpec{
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

			warnings, err := validator.ValidateUpdate(ctx, oldK8s, newK8s)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})

	Context("When validating K8s deletion", func() {
		It("should always allow deletion", func() {
			k8s := &kubecloudscalerv1alpha3.K8s{
				Spec: kubecloudscalerv1alpha3.K8sSpec{
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

			warnings, err := validator.ValidateDelete(ctx, k8s)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})
})
