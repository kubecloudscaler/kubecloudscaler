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

package service

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	flowtypes "github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/types"
)

var _ = Describe("ResourceCreatorService.buildPeriods", func() {
	var svc *ResourceCreatorService

	BeforeEach(func() {
		logger := zerolog.Nop()
		svc = NewResourceCreatorService(nil, nil, &logger)
	})

	It("rewrites recurring Start/End with the delayed times in HH:MM format", func() {
		base := common.ScalerPeriod{
			Name: "biz",
			Type: common.PeriodTypeUp,
			Time: common.TimePeriod{Recurring: &common.RecurringPeriod{
				StartTime: "09:00", EndTime: "17:00",
				Days: []common.DayOfWeek{common.DayAll},
			}},
		}
		delayed := flowtypes.PeriodWithDelay{
			Period:    base,
			StartTime: time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC),
			EndTime:   time.Date(0, 1, 1, 16, 45, 0, 0, time.UTC),
		}

		out := svc.buildPeriods([]flowtypes.PeriodWithDelay{delayed})

		Expect(out).To(HaveLen(1))
		Expect(out[0].Time.Recurring).ToNot(BeIdenticalTo(base.Time.Recurring)) // defensive copy
		Expect(out[0].Time.Recurring.StartTime).To(Equal("10:00"))
		Expect(out[0].Time.Recurring.EndTime).To(Equal("16:45"))
		// Source is not mutated
		Expect(base.Time.Recurring.StartTime).To(Equal("09:00"))
		Expect(base.Time.Recurring.EndTime).To(Equal("17:00"))
	})

	It("rewrites fixed Start/End with the delayed times in full datetime format", func() {
		base := common.ScalerPeriod{
			Name: "one-off",
			Type: common.PeriodTypeUp,
			Time: common.TimePeriod{Fixed: &common.FixedPeriod{
				StartTime: "2026-01-15 09:00:00",
				EndTime:   "2026-01-15 17:00:00",
			}},
		}
		delayed := flowtypes.PeriodWithDelay{
			Period:    base,
			StartTime: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2026, 1, 15, 16, 45, 0, 0, time.UTC),
		}

		out := svc.buildPeriods([]flowtypes.PeriodWithDelay{delayed})

		Expect(out).To(HaveLen(1))
		Expect(out[0].Time.Fixed.StartTime).To(Equal("2026-01-15 10:00:00"))
		Expect(out[0].Time.Fixed.EndTime).To(Equal("2026-01-15 16:45:00"))
		Expect(base.Time.Fixed.StartTime).To(Equal("2026-01-15 09:00:00"))
	})

	It("returns an empty slice (not nil) when given no periods", func() {
		out := svc.buildPeriods(nil)
		Expect(out).ToNot(BeNil())
		Expect(out).To(BeEmpty())
	})
})

var _ = Describe("ResourceCreatorService.CreateK8sResource and CreateGcpResource", func() {
	var (
		scheme *runtime.Scheme
		svc    *ResourceCreatorService
		flow   *kubecloudscalerv1alpha3.Flow
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())
		// Flow / K8s / Gcp are all cluster-scoped — no Namespace on the Flow.
		flow = &kubecloudscalerv1alpha3.Flow{
			ObjectMeta: metav1.ObjectMeta{Name: "demo-flow", UID: "flow-uid"},
		}
	})

	newSvcWithFlow := func() *ResourceCreatorService {
		logger := zerolog.Nop()
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(flow).Build()
		return NewResourceCreatorService(c, scheme, &logger)
	}

	It("creates a K8s child CR with labels, owner reference, and rewritten periods", func() {
		svc = newSvcWithFlow()
		base := common.ScalerPeriod{
			Name: "biz",
			Type: common.PeriodTypeUp,
			Time: common.TimePeriod{Recurring: &common.RecurringPeriod{
				StartTime: "09:00", EndTime: "17:00",
				Days: []common.DayOfWeek{common.DayAll},
			}},
		}
		periods := []flowtypes.PeriodWithDelay{{
			Period:    base,
			StartTime: time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC),
			EndTime:   time.Date(0, 1, 1, 16, 0, 0, 0, time.UTC),
		}}
		spec := kubecloudscalerv1alpha3.K8sResource{
			Name: "api",
			Resources: common.Resources{
				Names: []string{"api-deploy"},
			},
		}

		err := svc.CreateK8sResource(context.Background(), flow, "api", spec, periods)
		Expect(err).ToNot(HaveOccurred())

		var got kubecloudscalerv1alpha3.K8s
		Expect(svc.client.Get(context.Background(), types.NamespacedName{Name: "flow-demo-flow-api"}, &got)).To(Succeed())
		Expect(got.Labels).To(HaveKeyWithValue("flow", "demo-flow"))
		Expect(got.Labels).To(HaveKeyWithValue("resource", "api"))
		Expect(got.OwnerReferences).To(HaveLen(1))
		Expect(got.OwnerReferences[0].UID).To(Equal(flow.UID))
		Expect(*got.OwnerReferences[0].Controller).To(BeTrue())
		Expect(got.Spec.Resources.Names).To(ConsistOf("api-deploy"))
		Expect(got.Spec.Periods).To(HaveLen(1))
		Expect(got.Spec.Periods[0].Time.Recurring.StartTime).To(Equal("10:00"))
		Expect(got.Spec.Periods[0].Time.Recurring.EndTime).To(Equal("16:00"))
	})

	It("preserves existing labels/annotations and overwrites Spec on update", func() {
		svc = newSvcWithFlow()
		spec := kubecloudscalerv1alpha3.K8sResource{Name: "api"}

		Expect(svc.CreateK8sResource(context.Background(), flow, "api", spec, nil)).To(Succeed())

		// Simulate an external actor adding a label and annotation between reconciles.
		var existing kubecloudscalerv1alpha3.K8s
		Expect(svc.client.Get(context.Background(), types.NamespacedName{Name: "flow-demo-flow-api"}, &existing)).To(Succeed())
		if existing.Labels == nil {
			existing.Labels = map[string]string{}
		}
		if existing.Annotations == nil {
			existing.Annotations = map[string]string{}
		}
		existing.Labels["external"] = "kept"
		existing.Annotations["external"] = "kept"
		Expect(svc.client.Update(context.Background(), &existing)).To(Succeed())

		// Second call with a new Spec.
		spec.Resources.Names = []string{"api-v2"}
		Expect(svc.CreateK8sResource(context.Background(), flow, "api", spec, nil)).To(Succeed())

		var got kubecloudscalerv1alpha3.K8s
		Expect(svc.client.Get(context.Background(), types.NamespacedName{Name: "flow-demo-flow-api"}, &got)).To(Succeed())
		Expect(got.Labels).To(HaveKeyWithValue("external", "kept"))
		Expect(got.Labels).To(HaveKeyWithValue("flow", "demo-flow"))
		Expect(got.Annotations).To(HaveKeyWithValue("external", "kept"))
		Expect(got.Spec.Resources.Names).To(ConsistOf("api-v2"))
	})

	It("creates a GCP child CR with labels, owner reference, and matching spec", func() {
		svc = newSvcWithFlow()
		spec := kubecloudscalerv1alpha3.GcpResource{
			Name: "vm",
			Resources: common.Resources{
				Names: []string{"some-vm"},
			},
		}

		Expect(svc.CreateGcpResource(context.Background(), flow, "vm", spec, nil)).To(Succeed())

		var got kubecloudscalerv1alpha3.Gcp
		Expect(svc.client.Get(context.Background(), types.NamespacedName{Name: "flow-demo-flow-vm"}, &got)).To(Succeed())
		Expect(got.Labels).To(HaveKeyWithValue("flow", "demo-flow"))
		Expect(got.Labels).To(HaveKeyWithValue("resource", "vm"))
		Expect(got.OwnerReferences).To(HaveLen(1))
		Expect(got.Spec.Resources.Names).To(ConsistOf("some-vm"))
	})
})
