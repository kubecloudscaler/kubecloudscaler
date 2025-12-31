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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/k8s/service/handlers"
)

var _ = Describe("StatusHandler", func() {
	var (
		handler  service.Handler
		reconCtx *service.ReconciliationContext
		logger   zerolog.Logger
		scheme   *runtime.Scheme
		scaler   *kubecloudscalerv1alpha3.K8s
	)

	BeforeEach(func() {
		handler = handlers.NewStatusHandler()
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.K8s{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-scaler",
				Namespace: "default",
			},
			Status: common.ScalerStatus{
				CurrentPeriod: &common.ScalerStatusPeriod{},
			},
		}

		reconCtx = &service.ReconciliationContext{
			Request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-scaler",
					Namespace: "default",
				},
			},
			Client:         fake.NewClientBuilder().WithScheme(scheme).WithObjects(scaler).WithStatusSubresource(scaler).Build(),
			Logger:         &logger,
			Scaler:         scaler,
			SuccessResults: []common.ScalerStatusSuccess{},
			FailedResults:  []common.ScalerStatusFailed{},
		}
	})

	Context("When updating status with successful results", func() {
		It("should update status and set requeue", func() {
			reconCtx.SuccessResults = []common.ScalerStatusSuccess{
				{Kind: "deployment", Name: "test-deployment-1", Comment: "scaled up"},
			}

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Scaler.Status.CurrentPeriod.Successful).To(Equal(reconCtx.SuccessResults))
			Expect(reconCtx.RequeueAfter).To(BeNumerically(">", 0))
		})

		It("should complete in under 100ms", func() {
			startTime := time.Now()
			_ = handler.Execute(reconCtx)
			duration := time.Since(startTime)

			Expect(duration).To(BeNumerically("<", 100*time.Millisecond))
		})
	})

	Context("When updating status with failed results", func() {
		It("should update status with failures and set requeue", func() {
			reconCtx.FailedResults = []common.ScalerStatusFailed{
				{Kind: "deployment", Name: "test-deployment-2", Reason: "API error"},
			}

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Scaler.Status.CurrentPeriod.Failed).To(Equal(reconCtx.FailedResults))
		})
	})

	Context("When finalizer cleanup is requested", func() {
		It("should remove the finalizer and not set requeue", func() {
			controllerutil.AddFinalizer(reconCtx.Scaler, handlers.ScalerFinalizer)
			reconCtx.Client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(reconCtx.Scaler).Build()
			reconCtx.ShouldFinalize = true

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(controllerutil.ContainsFinalizer(reconCtx.Scaler, handlers.ScalerFinalizer)).To(BeFalse())
		})
	})

	Context("When both success and failed results exist", func() {
		It("should update status with both success and failures", func() {
			reconCtx.SuccessResults = []common.ScalerStatusSuccess{
				{Kind: "deployment", Name: "test-deployment-1", Comment: "scaled up"},
			}
			reconCtx.FailedResults = []common.ScalerStatusFailed{
				{Kind: "deployment", Name: "test-deployment-2", Reason: "API error"},
			}

			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.Scaler.Status.CurrentPeriod.Successful).To(Equal(reconCtx.SuccessResults))
			Expect(reconCtx.Scaler.Status.CurrentPeriod.Failed).To(Equal(reconCtx.FailedResults))
		})
	})

	Context("When this is the last handler in chain", func() {
		It("should not call next handler when next is nil", func() {
			// Default handler has no next set
			err := handler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
		})
	})
})
