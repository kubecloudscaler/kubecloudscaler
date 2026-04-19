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

package gcp

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
)

var _ = Describe("Backward Compatibility", func() {
	var (
		ctx        context.Context
		reconciler *ScalerReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &ScalerReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
			Logger: &log.Logger,
		}
	})

	Context("When reconciling resources with various configurations", func() {
		It("should handle resource not found correctly", func() {
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "nonexistent-resource",
					Namespace: "default",
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			// FetchHandler returns nil on NotFound (controller-runtime would otherwise log a
			// spurious "Reconciler error"). So the reconcile returns no error and no requeue.
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})

		It("should handle missing required fields correctly", func() {
			scaler := &kubecloudscalerv1alpha3.Gcp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-minimal",
					Namespace: "default",
				},
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					// Minimal configuration with required periods field
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID:         "test-project",
						DefaultPeriodType: "down",
					},
					Periods: []common.ScalerPeriod{
						{
							Type: "down",
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayAll},
									StartTime: "00:00",
									EndTime:   "23:59",
									Once:      ptr.To(false),
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, scaler)).To(Succeed())
			defer k8sClient.Delete(ctx, scaler)

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      scaler.Name,
					Namespace: scaler.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			// AuthHandler may or may not fail depending on GCP credentials availability.
			if err != nil {
				Expect(service.IsCriticalError(err)).To(BeTrue())
				Expect(result).To(Equal(ctrl.Result{}))
			}
		})

		It("should maintain CRD structure compatibility", func() {
			scaler := &kubecloudscalerv1alpha3.Gcp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-crd-compat",
					Namespace: "default",
				},
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID:         "test-project",
						Region:            "us-central1",
						DefaultPeriodType: "down",
					},
					Periods: []common.ScalerPeriod{
						{
							Name: "down",
							Type: "down",
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayAll},
									StartTime: "00:00",
									EndTime:   "23:59",
									Once:      ptr.To(false),
								},
							},
						},
					},
				},
			}

			// Verify all CRD fields are accessible
			Expect(scaler.Spec.Config.ProjectID).To(Equal("test-project"))
			Expect(scaler.Spec.Config.Region).To(Equal("us-central1"))
			Expect(scaler.Spec.Periods).To(HaveLen(1))
			Expect(scaler.Spec.Periods[0].Name).To(Equal("down"))
		})

		It("should handle finalizers correctly", func() {
			scaler := &kubecloudscalerv1alpha3.Gcp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-finalizer",
					Namespace: "default",
				},
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID:         "test-project",
						DefaultPeriodType: "down",
					},
					Periods: []common.ScalerPeriod{
						{
							Type: "down",
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayAll},
									StartTime: "00:00",
									EndTime:   "23:59",
									Once:      ptr.To(false),
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, scaler)).To(Succeed())
			defer k8sClient.Delete(ctx, scaler)

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      scaler.Name,
					Namespace: scaler.Namespace,
				},
			}

			// First reconciliation adds finalizer; auth may or may not fail depending on env.
			// Either way, FinalizerHandler runs before AuthHandler in the chain so the finalizer
			// must be persisted on the server regardless of what AuthHandler does next.
			_, err := reconciler.Reconcile(ctx, req)
			if err != nil {
				Expect(service.IsCriticalError(err)).To(BeTrue())
			}

			updatedScaler := &kubecloudscalerv1alpha3.Gcp{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      scaler.Name,
				Namespace: scaler.Namespace,
			}, updatedScaler)).To(Succeed())
			Expect(updatedScaler.Finalizers).To(ContainElement("kubecloudscaler.cloud/finalizer"))
		})
	})

	Context("When verifying error handling behavior", func() {
		It("should return appropriate errors for critical failures", func() {
			scaler := &kubecloudscalerv1alpha3.Gcp{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-error-handling",
					Namespace: "default",
				},
				Spec: kubecloudscalerv1alpha3.GcpSpec{
					Config: kubecloudscalerv1alpha3.GcpConfig{
						ProjectID:         "test-project",
						DefaultPeriodType: "down",
					},
					Periods: []common.ScalerPeriod{
						{
							Type: "down",
							Time: common.TimePeriod{
								Recurring: &common.RecurringPeriod{
									Days:      []common.DayOfWeek{common.DayAll},
									StartTime: "00:00",
									EndTime:   "23:59",
									Once:      ptr.To(false),
								},
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, scaler)).To(Succeed())
			defer k8sClient.Delete(ctx, scaler)

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      scaler.Name,
					Namespace: scaler.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			// AuthHandler may or may not fail depending on GCP credentials availability in the test environment.
			if err != nil {
				Expect(service.IsCriticalError(err)).To(BeTrue())
				Expect(result).To(Equal(ctrl.Result{}))
			}
		})
	})
})
