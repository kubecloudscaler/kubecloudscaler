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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
)

var _ = Describe("FinalizerHandler", func() {
	var (
		logger           zerolog.Logger
		scheme           *runtime.Scheme
		finalizerHandler service.Handler
		reconCtx         *service.ReconciliationContext
		scaler           *kubecloudscalerv1alpha3.Gcp
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.Gcp{}
		scaler.SetName("test-scaler")
		scaler.SetNamespace("default")

		finalizerHandler = handlers.NewFinalizerHandler()
	})

	Context("When scaler is not being deleted and has no finalizer", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should add finalizer and continue", func() {
			result, err := finalizerHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconCtx.ShouldFinalize).To(BeFalse())
		})
	})

	Context("When scaler is being deleted with finalizer", func() {
		BeforeEach(func() {
			now := metav1.Now()
			scaler.SetDeletionTimestamp(&now)
			scaler.SetFinalizers([]string{handlers.ScalerFinalizer})

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should set ShouldFinalize flag and continue", func() {
			result, err := finalizerHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconCtx.ShouldFinalize).To(BeTrue())
		})
	})

	Context("When scaler is being deleted without finalizer", func() {
		BeforeEach(func() {
			// Create without finalizer first
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			// Manually set deletion timestamp (simulating the scenario)
			now := metav1.Now()
			scaler.SetDeletionTimestamp(&now)

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should set SkipRemaining and stop chain", func() {
			result, err := finalizerHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconCtx.SkipRemaining).To(BeTrue())
		})
	})
})
