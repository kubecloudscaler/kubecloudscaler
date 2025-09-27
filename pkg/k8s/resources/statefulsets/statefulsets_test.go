package statefulsets_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	statefulsetsPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/statefulsets"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsV1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/utils/ptr"
)

var _ = Describe("StatefulSets", func() {
	var (
		ctx             context.Context
		statefulsets    *statefulsetsPkg.Statefulsets
		fakeClient      *fake.Clientset
		testNamespace   = "test-namespace"
		testStatefulSet = "test-statefulset"
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

			statefulsets = &statefulsetsPkg.Statefulsets{
				Resource: resource,
			}
			statefulsets.Client = fakeClient.AppsV1()
		})

		Context("when scaling down statefulsets", func() {
			BeforeEach(func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1

				// Create a test statefulset
				testStatefulSetObj := &appsV1.StatefulSet{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testStatefulSet,
						Namespace: testNamespace,
					},
					Spec: appsV1.StatefulSetSpec{
						Replicas: ptr.To(int32(3)),
					},
				}

				// Add the statefulset to the fake client
				_, err := fakeClient.AppsV1().StatefulSets(testNamespace).Create(ctx, testStatefulSetObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should scale down statefulset successfully", func() {
				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("statefulset"))
				Expect(success[0].Name).To(Equal(testStatefulSet))

				// Verify the statefulset was updated
				updatedStatefulSet, err := fakeClient.AppsV1().StatefulSets(testNamespace).Get(ctx, testStatefulSet, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedStatefulSet.Spec.Replicas).To(Equal(int32(1)))
			})

			It("should add annotations when scaling down", func() {
				success, _, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Verify annotations were added
				updatedStatefulSet, err := fakeClient.AppsV1().StatefulSets(testNamespace).Get(ctx, testStatefulSet, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedStatefulSet.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsOrigValue))
			})
		})

		Context("when scaling up statefulsets", func() {
			BeforeEach(func() {
				mockPeriod.Type = "up"
				mockPeriod.MaxReplicas = 5

				// Create a test statefulset
				testStatefulSetObj := &appsV1.StatefulSet{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testStatefulSet,
						Namespace: testNamespace,
					},
					Spec: appsV1.StatefulSetSpec{
						Replicas: ptr.To(int32(1)),
					},
				}

				// Add the statefulset to the fake client
				_, err := fakeClient.AppsV1().StatefulSets(testNamespace).Create(ctx, testStatefulSetObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should scale up statefulset successfully", func() {
				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("statefulset"))
				Expect(success[0].Name).To(Equal(testStatefulSet))

				// Verify the statefulset was updated
				updatedStatefulSet, err := fakeClient.AppsV1().StatefulSets(testNamespace).Get(ctx, testStatefulSet, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedStatefulSet.Spec.Replicas).To(Equal(int32(5)))
			})

			It("should add annotations when scaling up", func() {
				success, _, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Verify annotations were added
				updatedStatefulSet, err := fakeClient.AppsV1().StatefulSets(testNamespace).Get(ctx, testStatefulSet, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedStatefulSet.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsOrigValue))
			})
		})

		Context("when restoring statefulsets", func() {
			BeforeEach(func() {
				mockPeriod.Type = "restore"

				// Create a test statefulset with annotations
				testStatefulSetObj := &appsV1.StatefulSet{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testStatefulSet,
						Namespace: testNamespace,
						Annotations: map[string]string{
							"kubecloudscaler.cloud/original-value": "3",
						},
					},
					Spec: appsV1.StatefulSetSpec{
						Replicas: ptr.To(int32(1)),
					},
				}

				// Add the statefulset to the fake client
				_, err := fakeClient.AppsV1().StatefulSets(testNamespace).Create(ctx, testStatefulSetObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should restore statefulset successfully", func() {
				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("statefulset"))
				Expect(success[0].Name).To(Equal(testStatefulSet))

				// Verify the statefulset was restored
				updatedStatefulSet, err := fakeClient.AppsV1().StatefulSets(testNamespace).Get(ctx, testStatefulSet, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedStatefulSet.Spec.Replicas).To(Equal(int32(3)))
			})

			It("should handle already restored statefulsets", func() {
				// First restore
				success, _, err := statefulsets.SetState(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Second restore should not change anything since annotations are removed
				success, _, err = statefulsets.SetState(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0)) // No action needed
			})
		})

		Context("when handling errors", func() {
			It("should handle statefulset list error", func() {
				// Mock the fake client to return an error when listing statefulsets
				fakeClient.PrependReactor("list", "statefulsets", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("list error")
				})

				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error listing statefulsets"))
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(0))
			})

			It("should handle statefulset get error", func() {
				// Create a statefulset in the fake client
				testStatefulSetObj := &appsV1.StatefulSet{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testStatefulSet,
						Namespace: testNamespace,
					},
					Spec: appsV1.StatefulSetSpec{
						Replicas: ptr.To(int32(3)),
					},
				}

				_, err := fakeClient.AppsV1().StatefulSets(testNamespace).Create(ctx, testStatefulSetObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Mock the fake client to return an error when getting a specific statefulset
				fakeClient.PrependReactor("get", "statefulsets", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("get error")
				})

				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("statefulset"))
				Expect(failed[0].Name).To(Equal(testStatefulSet))
				Expect(failed[0].Reason).To(ContainSubstring("get error"))
			})

			It("should handle statefulset update error", func() {
				// Create a statefulset in the fake client
				testStatefulSetObj := &appsV1.StatefulSet{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testStatefulSet,
						Namespace: testNamespace,
					},
					Spec: appsV1.StatefulSetSpec{
						Replicas: ptr.To(int32(3)),
					},
				}

				_, err := fakeClient.AppsV1().StatefulSets(testNamespace).Create(ctx, testStatefulSetObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Mock the fake client to return an error when updating statefulsets
				fakeClient.PrependReactor("update", "statefulsets", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("update error")
				})

				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("statefulset"))
				Expect(failed[0].Name).To(Equal(testStatefulSet))
				Expect(failed[0].Reason).To(ContainSubstring("update error"))
			})

			It("should handle annotation restoration error", func() {
				mockPeriod.Type = "restore"

				// Create a statefulset with invalid annotations
				testStatefulSetObj := &appsV1.StatefulSet{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testStatefulSet,
						Namespace: testNamespace,
						Annotations: map[string]string{
							"kubecloudscaler.cloud/original-value": "invalid",
						},
					},
					Spec: appsV1.StatefulSetSpec{
						Replicas: ptr.To(int32(1)),
					},
				}

				_, err := fakeClient.AppsV1().StatefulSets(testNamespace).Create(ctx, testStatefulSetObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("statefulset"))
				Expect(failed[0].Name).To(Equal(testStatefulSet))
				Expect(failed[0].Reason).To(ContainSubstring("strconv.Atoi"))
			})
		})

		Context("with multiple statefulsets", func() {
			BeforeEach(func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1

				// Create multiple test statefulsets
				statefulsetNames := []string{"statefulset1", "statefulset2", "statefulset3"}
				for _, name := range statefulsetNames {
					testStatefulSetObj := &appsV1.StatefulSet{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      name,
							Namespace: testNamespace,
						},
						Spec: appsV1.StatefulSetSpec{
							Replicas: ptr.To(int32(3)),
						},
					}

					_, err := fakeClient.AppsV1().StatefulSets(testNamespace).Create(ctx, testStatefulSetObj, metaV1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should process all statefulsets", func() {
				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(3))
				Expect(failed).To(HaveLen(0))

				// Verify all statefulsets were scaled down
				for _, name := range []string{"statefulset1", "statefulset2", "statefulset3"} {
					statefulset, err := fakeClient.AppsV1().StatefulSets(testNamespace).Get(ctx, name, metaV1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(*statefulset.Spec.Replicas).To(Equal(int32(1)))
				}
			})

			It("should continue processing when one statefulset fails", func() {
				// Mock one statefulset to fail on update
				fakeClient.PrependReactor("update", "statefulsets", func(action testing.Action) (bool, runtime.Object, error) {
					updateAction := action.(testing.UpdateAction)
					if updateAction.GetObject().(*appsV1.StatefulSet).Name == "statefulset2" {
						return true, nil, errors.New("update error for statefulset2")
					}
					return false, nil, nil
				})

				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(2))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Name).To(Equal("statefulset2"))
			})
		})

		Context("with multiple namespaces", func() {
			var secondNamespace = "second-namespace"

			BeforeEach(func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1

				// Add second namespace
				statefulsets.Resource.NsList = []string{testNamespace, secondNamespace}

				// Create statefulsets in both namespaces
				for _, ns := range []string{testNamespace, secondNamespace} {
					testStatefulSetObj := &appsV1.StatefulSet{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      fmt.Sprintf("statefulset-%s", ns),
							Namespace: ns,
						},
						Spec: appsV1.StatefulSetSpec{
							Replicas: ptr.To(int32(3)),
						},
					}

					_, err := fakeClient.AppsV1().StatefulSets(ns).Create(ctx, testStatefulSetObj, metaV1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should process statefulsets from all namespaces", func() {
				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(2))
				Expect(failed).To(HaveLen(0))

				// Verify statefulsets in both namespaces were processed
				for _, ns := range []string{testNamespace, secondNamespace} {
					statefulset, err := fakeClient.AppsV1().StatefulSets(ns).Get(ctx, fmt.Sprintf("statefulset-%s", ns), metaV1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(*statefulset.Spec.Replicas).To(Equal(int32(1)))
				}
			})
		})

		Context("edge cases", func() {
			It("should handle statefulset with nil replicas", func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1

				// Create a statefulset with nil replicas (defaults to 1)
				testStatefulSetObj := &appsV1.StatefulSet{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testStatefulSet,
						Namespace: testNamespace,
					},
					Spec: appsV1.StatefulSetSpec{
						Replicas: nil, // This will default to 1
					},
				}

				_, err := fakeClient.AppsV1().StatefulSets(testNamespace).Create(ctx, testStatefulSetObj, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))

				// Verify the statefulset was updated
				updatedStatefulSet, err := fakeClient.AppsV1().StatefulSets(testNamespace).Get(ctx, testStatefulSet, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedStatefulSet.Spec.Replicas).To(Equal(int32(1)))
			})

			It("should handle empty namespace list", func() {
				statefulsets.Resource.NsList = []string{}

				success, failed, err := statefulsets.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(0))
			})
		})
	})
})
