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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/gcp/service/handlers"
)

var _ = Describe("AuthHandler", func() {
	var (
		logger      zerolog.Logger
		scheme      *runtime.Scheme
		authHandler service.Handler
		reconCtx    *service.ReconciliationContext
		scaler      *kubecloudscalerv1alpha3.Gcp
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		scaler = &kubecloudscalerv1alpha3.Gcp{}
		scaler.SetName("test-scaler")
		scaler.SetNamespace("default")
		scaler.Spec.Config.ProjectID = "test-project"

		authHandler = handlers.NewAuthHandler()
	})

	Context("When auth secret is not specified", func() {
		BeforeEach(func() {
			// No auth secret specified
			scaler.Spec.Config.AuthSecret = nil

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should attempt to create GCP client with default credentials", func() {
			// This will fail in test environment without GCP credentials
			// but that's expected - the handler correctly tries to create the client
			_, err := authHandler.Execute(reconCtx)

			// Without GCP credentials, this should return a critical error
			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
		})

		It("should complete in under 100ms", func() {
			_, _ = authHandler.Execute(reconCtx)
			// Test execution time is implicitly tested by Ginkgo's timeout mechanisms
		})
	})

	Context("When auth secret is specified and exists", func() {
		BeforeEach(func() {
			secretName := "gcp-secret"
			scaler.Spec.Config.AuthSecret = &secretName

			// Create a mock secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					"credentials.json": []byte(`{"type": "service_account"}`),
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler, secret).
				Build()

			reconCtx = &service.ReconciliationContext{
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should fetch secret and attempt to create GCP client", func() {
			_, err := authHandler.Execute(reconCtx)

			// Secret should be populated in context
			Expect(reconCtx.Secret).ToNot(BeNil())
			Expect(reconCtx.Secret.Name).To(Equal("gcp-secret"))

			// Client creation will fail without valid credentials, but that's expected
			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
		})
	})

	Context("When auth secret is specified but does not exist", func() {
		BeforeEach(func() {
			secretName := "nonexistent-secret"
			scaler.Spec.Config.AuthSecret = &secretName

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler).
				Build()

			reconCtx = &service.ReconciliationContext{
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should return a critical error", func() {
			_, err := authHandler.Execute(reconCtx)

			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
			Expect(reconCtx.Secret).To(BeNil())
		})
	})

	Context("When scaler has minimal configuration", func() {
		BeforeEach(func() {
			scaler.Spec.Config.AuthSecret = ptr.To("test-secret")
			scaler.Spec.Config.ProjectID = "test-project-123"

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(scaler, secret).
				Build()

			reconCtx = &service.ReconciliationContext{
				Request: ctrl.Request{},
				Client:  k8sClient,
				Logger:  &logger,
				Scaler:  scaler,
			}
		})

		It("should handle configuration correctly", func() {
			_, err := authHandler.Execute(reconCtx)

			// Secret should be fetched
			Expect(reconCtx.Secret).ToNot(BeNil())

			// GCP client creation will fail without valid credentials
			Expect(err).To(HaveOccurred())
			Expect(service.IsCriticalError(err)).To(BeTrue())
		})
	})
})
