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

var _ = Describe("ResourceMapperService.CreateResourceMappings", func() {
	var (
		logger zerolog.Logger
		svc    service.ResourceMapper
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		tc := service.NewTimeCalculatorService(&logger)
		svc = service.NewResourceMapperService(tc, &logger)
	})

	basePeriod := common.ScalerPeriod{
		Name: "biz",
		Type: common.PeriodTypeUp,
		Time: common.TimePeriod{Recurring: &common.RecurringPeriod{
			StartTime: "09:00", EndTime: "17:00",
			Days: []common.DayOfWeek{common.DayAll},
		}},
	}

	It("returns a ValidationError when a resource is declared in both K8s and GCP", func() {
		flow := &kubecloudscalerv1alpha3.Flow{
			Spec: kubecloudscalerv1alpha3.FlowSpec{
				Periods: []common.ScalerPeriod{basePeriod},
				Resources: kubecloudscalerv1alpha3.Resources{
					K8s: []kubecloudscalerv1alpha3.K8sResource{{Name: "api"}},
					Gcp: []kubecloudscalerv1alpha3.GcpResource{{Name: "api"}},
				},
				Flows: []kubecloudscalerv1alpha3.Flows{
					{PeriodName: "biz", Resources: []kubecloudscalerv1alpha3.FlowResource{{Name: "api"}}},
				},
			},
		}

		_, err := svc.CreateResourceMappings(flow, map[string]bool{"api": true})

		Expect(err).To(HaveOccurred())
		v, ok := service.AsValidationError(err)
		Expect(ok).To(BeTrue())
		Expect(v.Reason).To(Equal(service.ReasonAmbiguousResource))
	})

	It("returns a ValidationError when a resource is referenced but not defined", func() {
		flow := &kubecloudscalerv1alpha3.Flow{
			Spec: kubecloudscalerv1alpha3.FlowSpec{
				Periods: []common.ScalerPeriod{basePeriod},
				Flows: []kubecloudscalerv1alpha3.Flows{
					{PeriodName: "biz", Resources: []kubecloudscalerv1alpha3.FlowResource{{Name: "ghost"}}},
				},
			},
		}

		_, err := svc.CreateResourceMappings(flow, map[string]bool{"ghost": true})

		Expect(err).To(HaveOccurred())
		v, ok := service.AsValidationError(err)
		Expect(ok).To(BeTrue())
		Expect(v.Reason).To(Equal(service.ReasonUnknownResource))
	})

	It("returns a ValidationError when the same resource appears twice in a period", func() {
		flow := &kubecloudscalerv1alpha3.Flow{
			Spec: kubecloudscalerv1alpha3.FlowSpec{
				Periods: []common.ScalerPeriod{basePeriod},
				Resources: kubecloudscalerv1alpha3.Resources{
					K8s: []kubecloudscalerv1alpha3.K8sResource{{Name: "api"}},
				},
				Flows: []kubecloudscalerv1alpha3.Flows{
					{
						PeriodName: "biz",
						Resources: []kubecloudscalerv1alpha3.FlowResource{
							{Name: "api", StartTimeDelay: "0m"},
							{Name: "api", StartTimeDelay: "1m"},
						},
					},
				},
			},
		}

		_, err := svc.CreateResourceMappings(flow, map[string]bool{"api": true})

		Expect(err).To(HaveOccurred())
		v, ok := service.AsValidationError(err)
		Expect(ok).To(BeTrue())
		Expect(v.Reason).To(Equal(service.ReasonDuplicateResourceInPeriod))
	})

	It("produces a valid K8s mapping with associated periods", func() {
		flow := &kubecloudscalerv1alpha3.Flow{
			Spec: kubecloudscalerv1alpha3.FlowSpec{
				Periods: []common.ScalerPeriod{basePeriod},
				Resources: kubecloudscalerv1alpha3.Resources{
					K8s: []kubecloudscalerv1alpha3.K8sResource{{Name: "api"}},
				},
				Flows: []kubecloudscalerv1alpha3.Flows{
					{PeriodName: "biz", Resources: []kubecloudscalerv1alpha3.FlowResource{{Name: "api"}}},
				},
			},
		}

		mapping, err := svc.CreateResourceMappings(flow, map[string]bool{"api": true})

		Expect(err).ToNot(HaveOccurred())
		Expect(mapping).To(HaveKey("api"))
		Expect(mapping["api"].Type).To(Equal("k8s"))
		Expect(mapping["api"].K8sRes).ToNot(BeNil())
		Expect(mapping["api"].GcpRes).To(BeNil())
		Expect(mapping["api"].Periods).To(HaveLen(1))
	})
})
