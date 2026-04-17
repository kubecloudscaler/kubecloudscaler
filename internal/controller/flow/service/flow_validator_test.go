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

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service"
)

func newFlowWithPeriods(periods ...common.ScalerPeriod) *kubecloudscalerv1alpha3.Flow {
	return &kubecloudscalerv1alpha3.Flow{
		Spec: kubecloudscalerv1alpha3.FlowSpec{Periods: periods},
	}
}

var _ = Describe("FlowValidatorService.ValidatePeriodTimings", func() {
	var (
		logger zerolog.Logger
		svc    service.FlowValidator
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		tc := service.NewTimeCalculatorService(&logger)
		svc = service.NewFlowValidatorService(tc, &logger)
	})

	It("returns a ValidationError when a referenced period is not defined", func() {
		flow := newFlowWithPeriods() // no periods
		err := svc.ValidatePeriodTimings(flow, map[string]bool{"missing": true})

		Expect(err).To(HaveOccurred())
		Expect(service.IsValidationError(err)).To(BeTrue())
		v, _ := service.AsValidationError(err)
		Expect(v.Reason).To(Equal("UnknownPeriod"))
	})

	It("returns a ValidationError for an invalid delay format", func() {
		period := common.ScalerPeriod{
			Name: "biz",
			Type: common.PeriodTypeUp,
			Time: common.TimePeriod{Recurring: &common.RecurringPeriod{
				StartTime: "09:00", EndTime: "17:00",
				Days: []common.DayOfWeek{common.DayAll},
			}},
		}
		flow := newFlowWithPeriods(period)
		flow.Spec.Flows = []kubecloudscalerv1alpha3.Flows{
			{
				PeriodName: "biz",
				Resources: []kubecloudscalerv1alpha3.FlowResource{
					{Name: "api", StartTimeDelay: "not-a-duration"},
				},
			},
		}

		err := svc.ValidatePeriodTimings(flow, map[string]bool{"biz": true})

		Expect(err).To(HaveOccurred())
		Expect(service.IsValidationError(err)).To(BeTrue())
		v, _ := service.AsValidationError(err)
		Expect(v.Reason).To(Equal("InvalidDelayFormat"))
	})

	It("returns a ValidationError when delays invert the window", func() {
		// 1h period, startDelay=2h → adjustedDuration <= 0
		period := common.ScalerPeriod{
			Name: "short",
			Type: common.PeriodTypeUp,
			Time: common.TimePeriod{Recurring: &common.RecurringPeriod{
				StartTime: "09:00", EndTime: "10:00",
				Days: []common.DayOfWeek{common.DayAll},
			}},
		}
		flow := newFlowWithPeriods(period)
		flow.Spec.Flows = []kubecloudscalerv1alpha3.Flows{
			{
				PeriodName: "short",
				Resources: []kubecloudscalerv1alpha3.FlowResource{
					{Name: "api", StartTimeDelay: "2h"},
				},
			},
		}

		err := svc.ValidatePeriodTimings(flow, map[string]bool{"short": true})

		Expect(err).To(HaveOccurred())
		v, ok := service.AsValidationError(err)
		Expect(ok).To(BeTrue())
		Expect(v.Reason).To(Equal("InvertedWindow"))
	})

	It("returns nil for a valid flow with compatible delays", func() {
		period := common.ScalerPeriod{
			Name: "biz",
			Type: common.PeriodTypeUp,
			Time: common.TimePeriod{Recurring: &common.RecurringPeriod{
				StartTime: "09:00", EndTime: "17:00",
				Days: []common.DayOfWeek{common.DayAll},
			}},
		}
		flow := newFlowWithPeriods(period)
		flow.Spec.Flows = []kubecloudscalerv1alpha3.Flows{
			{
				PeriodName: "biz",
				Resources: []kubecloudscalerv1alpha3.FlowResource{
					{Name: "api", StartTimeDelay: "30m", EndTimeDelay: "15m"},
				},
			},
		}

		Expect(svc.ValidatePeriodTimings(flow, map[string]bool{"biz": true})).To(Succeed())
	})
})
