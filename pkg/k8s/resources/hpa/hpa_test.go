package hpa_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	autoscaleV2 "k8s.io/api/autoscaling/v2"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/hpa"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

var _ = Describe("HPA", func() {
	var (
		ctx           context.Context
		hpaResource   *hpa.HorizontalPodAutoscalers
		fakeClient    *fake.Clientset
		testNamespace = "test-namespace"
		testHPA       = "test-hpa"
	)

	BeforeEach(func() {
		ctx = context.Background()
		fakeClient = fake.NewSimpleClientset()
	})

	Describe("SetState", func() {
		var (
			mockPeriod *period.Period
			resource   *utils.K8sResource
		)

		BeforeEach(func() {
			mockPeriod = &period.Period{
				Type:         "down",
				MinReplicas:  1,
				MaxReplicas:  5,
				IsActive:     true,
				GetStartTime: time.Now(),
				GetEndTime:   time.Now(),
				Period: &common.RecurringPeriod{
					Days:      []string{"all"},
					StartTime: "00:00",
					EndTime:   "23:59",
				},
			}

			resource = &utils.K8sResource{
				NsList:      []string{testNamespace},
				ListOptions: metaV1.ListOptions{},
				Period:      mockPeriod,
			}

			hpaResource = &hpa.HorizontalPodAutoscalers{
				Resource: resource,
				Logger:   &log.Logger,
			}
			hpaResource.Client = fakeClient.AutoscalingV2()
		})

		Context("when scaling down HPAs", func() {
			BeforeEach(func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1
				mockPeriod.MaxReplicas = 3

				// Create a test HPA
				testHPAObj := &autoscaleV2.HorizontalPodAutoscaler{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testHPA,
						Namespace: testNamespace,
					},
					Spec: autoscaleV2.HorizontalPodAutoscalerSpec{
						MinReplicas: ptr.To(int32(3)),
						MaxReplicas: int32(10),
					},
				}

				// Add the HPA to the fake client
				_, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(ctx, testHPAObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should scale down HPA successfully", func() {
				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("hpa"))
				Expect(success[0].Name).To(Equal(testHPA))

				// Verify the HPA was updated
				updatedHPA, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Get(ctx, testHPA, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedHPA.Spec.MinReplicas).To(Equal(int32(1)))
				Expect(updatedHPA.Spec.MaxReplicas).To(Equal(int32(3)))
			})

			It("should add annotations when scaling down", func() {
				success, _, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Verify annotations were added
				updatedHPA, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Get(ctx, testHPA, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedHPA.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue))
				Expect(updatedHPA.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMaxOrigValue))
			})
		})

		Context("when scaling up HPAs", func() {
			BeforeEach(func() {
				mockPeriod.Type = "up"
				mockPeriod.MinReplicas = 5
				mockPeriod.MaxReplicas = 15

				// Create a test HPA
				testHPAObj := &autoscaleV2.HorizontalPodAutoscaler{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testHPA,
						Namespace: testNamespace,
					},
					Spec: autoscaleV2.HorizontalPodAutoscalerSpec{
						MinReplicas: ptr.To(int32(1)),
						MaxReplicas: int32(5),
					},
				}

				// Add the HPA to the fake client
				_, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(ctx, testHPAObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should scale up HPA successfully", func() {
				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("hpa"))
				Expect(success[0].Name).To(Equal(testHPA))

				// Verify the HPA was updated
				updatedHPA, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Get(ctx, testHPA, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedHPA.Spec.MinReplicas).To(Equal(int32(5)))
				Expect(updatedHPA.Spec.MaxReplicas).To(Equal(int32(15)))
			})

			It("should add annotations when scaling up", func() {
				success, _, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Verify annotations were added
				updatedHPA, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Get(ctx, testHPA, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedHPA.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue))
				Expect(updatedHPA.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMaxOrigValue))
			})
		})

		Context("when restoring HPAs", func() {
			BeforeEach(func() {
				mockPeriod.Type = "restore"

				// Create a test HPA with annotations
				testHPAObj := &autoscaleV2.HorizontalPodAutoscaler{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testHPA,
						Namespace: testNamespace,
						Annotations: map[string]string{
							"kubecloudscaler.cloud/min-original-value": "3",
							"kubecloudscaler.cloud/max-original-value": "10",
						},
					},
					Spec: autoscaleV2.HorizontalPodAutoscalerSpec{
						MinReplicas: ptr.To(int32(1)),
						MaxReplicas: int32(5),
					},
				}

				// Add the HPA to the fake client
				_, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(ctx, testHPAObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should restore HPA successfully", func() {
				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("hpa"))
				Expect(success[0].Name).To(Equal(testHPA))

				// Verify the HPA was restored
				updatedHPA, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Get(ctx, testHPA, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedHPA.Spec.MinReplicas).To(Equal(int32(3)))
				Expect(updatedHPA.Spec.MaxReplicas).To(Equal(int32(10)))
			})

			It("should handle already restored HPAs", func() {
				// First restore
				success, _, err := hpaResource.SetState(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Second restore should not change anything since annotations are removed
				success, _, err = hpaResource.SetState(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0)) // No action needed
			})
		})

		Context("when handling errors", func() {
			It("should handle HPA list error", func() {
				// Mock the fake client to return an error when listing HPAs
				fakeClient.PrependReactor("list", "horizontalpodautoscalers", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("list error")
				})

				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error listing hpas"))
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(0))
			})

			It("should handle HPA get error", func() {
				// Create an HPA in the fake client
				testHPAObj := &autoscaleV2.HorizontalPodAutoscaler{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testHPA,
						Namespace: testNamespace,
					},
					Spec: autoscaleV2.HorizontalPodAutoscalerSpec{
						MinReplicas: ptr.To(int32(3)),
						MaxReplicas: int32(10),
					},
				}

				_, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(ctx, testHPAObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Mock the fake client to return an error when getting a specific HPA
				fakeClient.PrependReactor("get", "horizontalpodautoscalers", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("get error")
				})

				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("hpa"))
				Expect(failed[0].Name).To(Equal(testHPA))
				Expect(failed[0].Reason).To(ContainSubstring("get error"))
			})

			It("should handle HPA update error", func() {
				// Create an HPA in the fake client
				testHPAObj := &autoscaleV2.HorizontalPodAutoscaler{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testHPA,
						Namespace: testNamespace,
					},
					Spec: autoscaleV2.HorizontalPodAutoscalerSpec{
						MinReplicas: ptr.To(int32(3)),
						MaxReplicas: int32(10),
					},
				}

				_, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(ctx, testHPAObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Mock the fake client to return an error when updating HPAs
				fakeClient.PrependReactor("update", "horizontalpodautoscalers", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("update error")
				})

				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("hpa"))
				Expect(failed[0].Name).To(Equal(testHPA))
				Expect(failed[0].Reason).To(ContainSubstring("update error"))
			})

			It("should handle annotation restoration error", func() {
				mockPeriod.Type = "restore"

				// Create an HPA with invalid annotations
				testHPAObj := &autoscaleV2.HorizontalPodAutoscaler{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testHPA,
						Namespace: testNamespace,
						Annotations: map[string]string{
							"kubecloudscaler.cloud/min-original-value": "invalid",
						},
					},
					Spec: autoscaleV2.HorizontalPodAutoscalerSpec{
						MinReplicas: ptr.To(int32(1)),
						MaxReplicas: int32(5),
					},
				}

				_, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(ctx, testHPAObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("hpa"))
				Expect(failed[0].Name).To(Equal(testHPA))
				Expect(failed[0].Reason).To(ContainSubstring("strconv.Atoi"))
			})
		})

		Context("with multiple HPAs", func() {
			BeforeEach(func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1
				mockPeriod.MaxReplicas = 3

				// Create multiple test HPAs
				hpaNames := []string{"hpa1", "hpa2", "hpa3"}
				for _, name := range hpaNames {
					testHPAObj := &autoscaleV2.HorizontalPodAutoscaler{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      name,
							Namespace: testNamespace,
						},
						Spec: autoscaleV2.HorizontalPodAutoscalerSpec{
							MinReplicas: ptr.To(int32(3)),
							MaxReplicas: int32(10),
						},
					}

					_, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(ctx, testHPAObj, metaV1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should process all HPAs", func() {
				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(3))
				Expect(failed).To(HaveLen(0))

				// Verify all HPAs were scaled down
				for _, name := range []string{"hpa1", "hpa2", "hpa3"} {
					hpaObj, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Get(ctx, name, metaV1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(*hpaObj.Spec.MinReplicas).To(Equal(int32(1)))
					Expect(hpaObj.Spec.MaxReplicas).To(Equal(int32(3)))
				}
			})

			It("should continue processing when one HPA fails", func() {
				// Mock one HPA to fail on update
				fakeClient.PrependReactor("update", "horizontalpodautoscalers", func(action testing.Action) (bool, runtime.Object, error) {
					updateAction := action.(testing.UpdateAction)
					if updateAction.GetObject().(*autoscaleV2.HorizontalPodAutoscaler).Name == "hpa2" {
						return true, nil, errors.New("update error for hpa2")
					}
					return false, nil, nil
				})

				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(2))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Name).To(Equal("hpa2"))
			})
		})

		Context("with multiple namespaces", func() {
			var secondNamespace = "second-namespace"

			BeforeEach(func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1
				mockPeriod.MaxReplicas = 3

				// Add second namespace
				hpaResource.Resource.NsList = []string{testNamespace, secondNamespace}

				// Create HPAs in both namespaces
				for _, ns := range []string{testNamespace, secondNamespace} {
					testHPAObj := &autoscaleV2.HorizontalPodAutoscaler{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      fmt.Sprintf("hpa-%s", ns),
							Namespace: ns,
						},
						Spec: autoscaleV2.HorizontalPodAutoscalerSpec{
							MinReplicas: ptr.To(int32(3)),
							MaxReplicas: int32(10),
						},
					}

					_, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(ns).Create(ctx, testHPAObj, metaV1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should process HPAs from all namespaces", func() {
				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(2))
				Expect(failed).To(HaveLen(0))

				// Verify HPAs in both namespaces were processed
				for _, ns := range []string{testNamespace, secondNamespace} {
					hpaObj, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(ns).Get(ctx, fmt.Sprintf("hpa-%s", ns), metaV1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(*hpaObj.Spec.MinReplicas).To(Equal(int32(1)))
					Expect(hpaObj.Spec.MaxReplicas).To(Equal(int32(3)))
				}
			})
		})

		Context("edge cases", func() {
			It("should handle HPA with nil minReplicas", func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1
				mockPeriod.MaxReplicas = 3

				// Create an HPA with nil minReplicas (defaults to 1)
				testHPAObj := &autoscaleV2.HorizontalPodAutoscaler{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testHPA,
						Namespace: testNamespace,
					},
					Spec: autoscaleV2.HorizontalPodAutoscalerSpec{
						MinReplicas: nil, // This will default to 1
						MaxReplicas: int32(10),
					},
				}

				_, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Create(ctx, testHPAObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))

				// Verify the HPA was updated
				updatedHPA, err := fakeClient.AutoscalingV2().HorizontalPodAutoscalers(testNamespace).Get(ctx, testHPA, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedHPA.Spec.MinReplicas).To(Equal(int32(1)))
				Expect(updatedHPA.Spec.MaxReplicas).To(Equal(int32(3)))
			})

			It("should handle empty namespace list", func() {
				hpaResource.Resource.NsList = []string{}

				success, failed, err := hpaResource.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(0))
			})
		})
	})
})
