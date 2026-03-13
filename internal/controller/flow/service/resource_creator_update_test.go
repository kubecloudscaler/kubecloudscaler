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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

var _ = Describe("ResourceCreatorService", func() {
	var (
		scheme    *runtime.Scheme
		logger    zerolog.Logger
		svc       *ResourceCreatorService
		k8sClient client.Client
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(kubecloudscalerv1alpha3.AddToScheme(scheme)).To(Succeed())

		logger = zerolog.Nop()
		k8sClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&kubecloudscalerv1alpha3.K8s{}).
			Build()
		svc = NewResourceCreatorService(k8sClient, scheme, &logger)
	})

	Describe("createOrUpdateResource", func() {
		Context("when creating a new resource", func() {
			It("should create the resource successfully", func() {
				obj := &kubecloudscalerv1alpha3.K8s{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-k8s-create",
						Namespace: "default",
					},
					Spec: kubecloudscalerv1alpha3.K8sSpec{
						DryRun: false,
					},
				}

				err := svc.createOrUpdateResource(context.Background(), obj)
				Expect(err).ToNot(HaveOccurred())

				var createdObj kubecloudscalerv1alpha3.K8s
				err = k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      "test-k8s-create",
					Namespace: "default",
				}, &createdObj)

				Expect(err).ToNot(HaveOccurred())
				Expect(createdObj.Name).To(Equal("test-k8s-create"))
			})
		})

		Context("when updating an existing resource", func() {
			It("should update the spec and merge labels and annotations", func() {
				initialObj := &kubecloudscalerv1alpha3.K8s{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-k8s-update",
						Namespace: "default",
						Labels: map[string]string{
							"original": "label",
						},
						Annotations: map[string]string{
							"original": "annotation",
						},
					},
					Spec: kubecloudscalerv1alpha3.K8sSpec{
						DryRun: false,
					},
				}

				err := k8sClient.Create(context.Background(), initialObj)
				Expect(err).ToNot(HaveOccurred())

				updatedObj := &kubecloudscalerv1alpha3.K8s{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-k8s-update",
						Namespace: "default",
						Labels: map[string]string{
							"new": "label",
						},
						Annotations: map[string]string{
							"new": "annotation",
						},
					},
					Spec: kubecloudscalerv1alpha3.K8sSpec{
						DryRun: true,
					},
				}

				err = svc.createOrUpdateResource(context.Background(), updatedObj)
				Expect(err).ToNot(HaveOccurred())

				var finalObj kubecloudscalerv1alpha3.K8s
				err = k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      "test-k8s-update",
					Namespace: "default",
				}, &finalObj)

				Expect(err).ToNot(HaveOccurred())
				Expect(finalObj.Spec.DryRun).To(BeTrue())
				Expect(finalObj.Labels).To(HaveKeyWithValue("original", "label"))
				Expect(finalObj.Labels).To(HaveKeyWithValue("new", "label"))
				Expect(finalObj.Annotations).To(HaveKeyWithValue("original", "annotation"))
				Expect(finalObj.Annotations).To(HaveKeyWithValue("new", "annotation"))
			})
		})

		Context("when updating with new labels and annotations", func() {
			It("should preserve existing metadata and add new entries", func() {
				initialObj := &kubecloudscalerv1alpha3.K8s{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-k8s-labels",
						Namespace: "default",
						Labels: map[string]string{
							"existing": "label",
						},
						Annotations: map[string]string{
							"existing": "annotation",
						},
					},
					Spec: kubecloudscalerv1alpha3.K8sSpec{
						DryRun: false,
					},
				}

				err := k8sClient.Create(context.Background(), initialObj)
				Expect(err).ToNot(HaveOccurred())

				updatedObj := &kubecloudscalerv1alpha3.K8s{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-k8s-labels",
						Namespace: "default",
						Labels: map[string]string{
							"existing": "label",
							"new":      "label",
						},
						Annotations: map[string]string{
							"existing": "annotation",
							"new":      "annotation",
						},
					},
					Spec: kubecloudscalerv1alpha3.K8sSpec{
						DryRun: true,
					},
				}

				err = svc.createOrUpdateResource(context.Background(), updatedObj)
				Expect(err).ToNot(HaveOccurred())

				var finalObj kubecloudscalerv1alpha3.K8s
				err = k8sClient.Get(context.Background(), types.NamespacedName{
					Name:      "test-k8s-labels",
					Namespace: "default",
				}, &finalObj)

				Expect(err).ToNot(HaveOccurred())
				Expect(finalObj.Spec.DryRun).To(BeTrue())
				Expect(finalObj.Labels).To(HaveKeyWithValue("existing", "label"))
				Expect(finalObj.Labels).To(HaveKeyWithValue("new", "label"))
				Expect(finalObj.Annotations).To(HaveKeyWithValue("existing", "annotation"))
				Expect(finalObj.Annotations).To(HaveKeyWithValue("new", "annotation"))
			})
		})
	})
})
