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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
)

var _ = Describe("StatusHandler", func() {
	var (
		logger        zerolog.Logger
		scheme        *runtime.Scheme
		statusHandler service.Handler
		reconCtx      *service.ReconciliationContext
		scaler        *kubecloudscalerv1alpha3.Gcp
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.Gcp{}
		scaler.SetName("test-scaler")
		scaler.SetNamespace("default")

		statusHandler = handlers.NewStatusHandler()
	})

	Context("When updating status with successful results", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
				SuccessResults: []common.ScalerStatusSuccess{
					{Name: "instance-1", Kind: "instance"},
				},
				FailedResults: []common.ScalerStatusFailed{},
			}
		})

		It("should update status successfully", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileSuccessDuration))
		})

	})

	Context("When updating status with failed results", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				SuccessResults: []common.ScalerStatusSuccess{},
				FailedResults: []common.ScalerStatusFailed{
					{Name: "instance-2", Kind: "instance", Reason: "API error"},
				},
			}
		})

		It("should update status with failures", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileSuccessDuration))
		})
	})

	Context("When handling finalizer cleanup", func() {
		BeforeEach(func() {
			// Add finalizer to scaler
			scaler.SetFinalizers([]string{handlers.ScalerFinalizer})

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				ShouldFinalize: true, // Deletion in progress
			}
		})

		It("should remove finalizer", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When client Patch fails during finalizer removal", func() {
		BeforeEach(func() {
			scaler.SetFinalizers([]string{handlers.ScalerFinalizer})

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithInterceptorFuncs(interceptor.Funcs{
					Patch: func(
						ctx context.Context,
						c client.WithWatch,
						obj client.Object,
						patch client.Patch,
						opts ...client.PatchOption,
					) error {
						return fmt.Errorf("persistent patch failure")
					},
				}).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				ShouldFinalize: true,
			}
		})

		It("should return a recoverable error and set RequeueAfter", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(Equal(5 * time.Second))
		})
	})

	Context("When client Status Update fails", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				WithInterceptorFuncs(interceptor.Funcs{
					SubResourceUpdate: func(
						ctx context.Context,
						c client.Client,
						subResourceName string,
						obj client.Object,
						opts ...client.SubResourceUpdateOption,
					) error {
						return fmt.Errorf("status update failure")
					},
				}).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:            context.Background(),
				Request:        ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:         k8sClient,
				Logger:         &logger,
				Scaler:         scaler,
				SuccessResults: []common.ScalerStatusSuccess{},
				FailedResults:  []common.ScalerStatusFailed{},
			}
		})

		It("should return a recoverable error and set RequeueAfter", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsRecoverableError(err)).To(BeTrue())
			Expect(reconCtx.RequeueAfter).To(Equal(5 * time.Second))
		})
	})

	Context("When both success and failed results exist", func() {
		BeforeEach(func() {
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				WithStatusSubresource(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Ctx:     context.Background(),
				Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-scaler", Namespace: "default"}},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
				SuccessResults: []common.ScalerStatusSuccess{
					{Name: "instance-1", Kind: "instance"},
					{Name: "instance-3", Kind: "instance"},
				},
				FailedResults: []common.ScalerStatusFailed{
					{Name: "instance-2", Kind: "instance", Reason: "timeout"},
				},
			}
		})

		It("should update status with both success and failures", func() {
			err := statusHandler.Execute(reconCtx)

			Expect(err).ToNot(HaveOccurred())
			Expect(reconCtx.RequeueAfter).To(Equal(utils.ReconcileSuccessDuration))
		})
	})
})
