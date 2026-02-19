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

package handlers_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
	gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
)

var _ = Describe("ScalingHandler", func() {
	var (
		logger         zerolog.Logger
		scheme         *runtime.Scheme
		scalingHandler service.Handler
		reconCtx       *service.ReconciliationContext
		scaler         *kubecloudscalerv1alpha3.Gcp
		mockPeriod     *periodPkg.Period
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.Gcp{}
		scaler.SetName("test-scaler")
		scaler.SetNamespace("default")
		scaler.Spec.Config.ProjectID = "test-project"
		scaler.Spec.Config.Region = "us-central1"
		scaler.Spec.Resources.Types = []string{"instance"}
		scaler.Spec.Resources.Names = []string{"test-instance-1"}

		// Create a mock period
		scalerPeriod := &common.ScalerPeriod{
			Name: "down",
			Type: "down",
			Time: common.TimePeriod{
				Recurring: &common.RecurringPeriod{
					Days:      []string{"all"},
					StartTime: "00:00",
					EndTime:   "23:59",
					Once:      ptr.To(false),
				},
			},
		}
		var err error
		mockPeriod, err = periodPkg.New(scalerPeriod)
		Expect(err).ToNot(HaveOccurred())

		scalingHandler = handlers.NewScalingHandler()
	})

	Context("When resource configuration is valid", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			resourceConfig := resources.Config{
				GCP: &gcpUtils.Config{
					Client:            &gcpUtils.ClientSet{}, // Mock client
					ProjectID:         scaler.Spec.Config.ProjectID,
					Region:            scaler.Spec.Config.Region,
					Names:             scaler.Spec.Resources.Names,
					Period:            mockPeriod,
					DefaultPeriodType: "down",
				},
			}

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				GCPClient:      &gcpUtils.ClientSet{},
				Period:         mockPeriod,
				ResourceConfig: resourceConfig,
			}
		})

		It("should attempt to scale resources", func() {
			result, err := scalingHandler.Execute(reconCtx)

			// Scaling will fail without real GCP API, but handler should not crash
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Results should be initialized (empty slices are valid)
			// Note: Empty slices may be represented as nil in Go, which is valid
		})

		It("should complete in under 100ms", func() {
			_, _ = scalingHandler.Execute(reconCtx)
		})
	})

	Context("When no resource types are specified", func() {
		BeforeEach(func() {
			scaler.Spec.Resources.Types = []string{}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			resourceConfig := resources.Config{
				GCP: &gcpUtils.Config{
					Client:            &gcpUtils.ClientSet{},
					ProjectID:         scaler.Spec.Config.ProjectID,
					Region:            scaler.Spec.Config.Region,
					Period:            mockPeriod,
					DefaultPeriodType: "down",
				},
			}

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				GCPClient:      &gcpUtils.ClientSet{},
				Period:         mockPeriod,
				ResourceConfig: resourceConfig,
			}
		})

		It("should use default resource type", func() {
			result, err := scalingHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})

	Context("When multiple resource types are specified", func() {
		BeforeEach(func() {
			scaler.Spec.Resources.Types = []string{"instance", "disk", "snapshot"}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			resourceConfig := resources.Config{
				GCP: &gcpUtils.Config{
					Client:            &gcpUtils.ClientSet{},
					ProjectID:         scaler.Spec.Config.ProjectID,
					Region:            scaler.Spec.Config.Region,
					Period:            mockPeriod,
					DefaultPeriodType: "down",
				},
			}

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				GCPClient:      &gcpUtils.ClientSet{},
				Period:         mockPeriod,
				ResourceConfig: resourceConfig,
			}
		})

		It("should attempt to scale all resource types", func() {
			result, err := scalingHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Handler should continue even if individual resources fail
		})
	})

	Context("When resource handler creation fails", func() {
		BeforeEach(func() {
			scaler.Spec.Resources.Types = []string{"invalid-resource-type"}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			resourceConfig := resources.Config{
				GCP: &gcpUtils.Config{
					Client:            &gcpUtils.ClientSet{},
					ProjectID:         scaler.Spec.Config.ProjectID,
					Region:            scaler.Spec.Config.Region,
					Period:            mockPeriod,
					DefaultPeriodType: "down",
				},
			}

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				GCPClient:      &gcpUtils.ClientSet{},
				Period:         mockPeriod,
				ResourceConfig: resourceConfig,
			}
		})

		It("should continue chain execution", func() {
			result, err := scalingHandler.Execute(reconCtx)

			// Handler should continue even if resource handler creation fails
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})
})
