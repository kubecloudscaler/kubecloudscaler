package cronjobs_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	cronjobsPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/cronjobs"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

var _ = Describe("Cronjobs", func() {
	var (
		ctx           context.Context
		cronjobs      *cronjobsPkg.Cronjobs
		fakeClient    *fake.Clientset
		testNamespace = "test-namespace"
		testCronjob   = "test-cronjob"
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
				MinReplicas:  0,
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

			cronjobs = &cronjobsPkg.Cronjobs{
				Resource: resource,
				Logger:   &log.Logger,
			}
			cronjobs.Client = fakeClient.BatchV1()
		})

		Context("when scaling down cronjobs", func() {
			BeforeEach(func() {
				mockPeriod.Type = "down"

				// Create a test cronjob
				testCronJob := &batchV1.CronJob{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testCronjob,
						Namespace: testNamespace,
					},
					Spec: batchV1.CronJobSpec{
						Suspend: ptr.To(false),
					},
				}

				// Add the cronjob to the fake client
				_, err := fakeClient.BatchV1().CronJobs(testNamespace).Create(ctx, testCronJob, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should suspend cronjob successfully", func() {
				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("cronjob"))
				Expect(success[0].Name).To(Equal(testCronjob))

				// Verify the cronjob was suspended
				updatedCronJob, err := fakeClient.BatchV1().CronJobs(testNamespace).Get(ctx, testCronjob, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedCronJob.Spec.Suspend).To(Equal(true))
			})

			It("should add annotations when suspending", func() {
				success, _, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Verify annotations were added
				updatedCronJob, err := fakeClient.BatchV1().CronJobs(testNamespace).Get(ctx, testCronjob, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedCronJob.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsOrigValue))
				Expect(updatedCronJob.Annotations[utils.AnnotationsPrefix+"/"+utils.AnnotationsOrigValue]).To(Equal("false"))
			})
		})

		Context("when scaling up cronjobs", func() {
			BeforeEach(func() {
				mockPeriod.Type = "up"

				// Create a test cronjob
				testCronJob := &batchV1.CronJob{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testCronjob,
						Namespace: testNamespace,
					},
					Spec: batchV1.CronJobSpec{
						Suspend: ptr.To(true),
					},
				}

				// Add the cronjob to the fake client
				_, err := fakeClient.BatchV1().CronJobs(testNamespace).Create(ctx, testCronJob, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return error for scaling up", func() {
				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("cronjob"))
				Expect(failed[0].Name).To(Equal(testCronjob))
				Expect(failed[0].Reason).To(Equal("cronjob can only be scaled down"))
			})
		})

		Context("when restoring cronjobs", func() {
			BeforeEach(func() {
				mockPeriod.Type = "restore"

				// Create a test cronjob with annotations
				testCronJob := &batchV1.CronJob{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testCronjob,
						Namespace: testNamespace,
						Annotations: map[string]string{
							"kubecloudscaler.cloud/original-value": "false",
						},
					},
					Spec: batchV1.CronJobSpec{
						Suspend: ptr.To(true),
					},
				}

				// Add the cronjob to the fake client
				_, err := fakeClient.BatchV1().CronJobs(testNamespace).Create(ctx, testCronJob, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should restore cronjob successfully", func() {
				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("cronjob"))
				Expect(success[0].Name).To(Equal(testCronjob))

				// Verify the cronjob was restored
				updatedCronJob, err := fakeClient.BatchV1().CronJobs(testNamespace).Get(ctx, testCronjob, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedCronJob.Spec.Suspend).To(Equal(false))
			})

			It("should handle already restored cronjobs", func() {
				// First restore
				success, _, err := cronjobs.SetState(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Second restore should not change anything since annotations are removed
				success, _, err = cronjobs.SetState(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0)) // No action needed
			})
		})

		Context("when handling errors", func() {
			It("should handle cronjob list error", func() {
				// Mock the fake client to return an error when listing cronjobs
				fakeClient.PrependReactor("list", "cronjobs", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("list error")
				})

				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error listing cronjobs"))
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(0))
			})

			It("should handle cronjob get error", func() {
				// Create a cronjob in the fake client
				testCronJob := &batchV1.CronJob{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testCronjob,
						Namespace: testNamespace,
					},
					Spec: batchV1.CronJobSpec{
						Suspend: ptr.To(false),
					},
				}

				_, err := fakeClient.BatchV1().CronJobs(testNamespace).Create(ctx, testCronJob, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Mock the fake client to return an error when getting a specific cronjob
				fakeClient.PrependReactor("get", "cronjobs", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("get error")
				})

				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("cronjob"))
				Expect(failed[0].Name).To(Equal(testCronjob))
				Expect(failed[0].Reason).To(ContainSubstring("get error"))
			})

			It("should handle cronjob update error", func() {
				// Create a cronjob in the fake client
				testCronJob := &batchV1.CronJob{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testCronjob,
						Namespace: testNamespace,
					},
					Spec: batchV1.CronJobSpec{
						Suspend: ptr.To(false),
					},
				}

				_, err := fakeClient.BatchV1().CronJobs(testNamespace).Create(ctx, testCronJob, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Mock the fake client to return an error when updating cronjobs
				fakeClient.PrependReactor("update", "cronjobs", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("update error")
				})

				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("cronjob"))
				Expect(failed[0].Name).To(Equal(testCronjob))
				Expect(failed[0].Reason).To(ContainSubstring("update error"))
			})

			It("should handle annotation restoration error", func() {
				mockPeriod.Type = "restore"

				// Create a cronjob with invalid annotations
				testCronJob := &batchV1.CronJob{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testCronjob,
						Namespace: testNamespace,
						Annotations: map[string]string{
							"kubecloudscaler.cloud/original-value": "invalid",
						},
					},
					Spec: batchV1.CronJobSpec{
						Suspend: ptr.To(true),
					},
				}

				_, err := fakeClient.BatchV1().CronJobs(testNamespace).Create(ctx, testCronJob, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("cronjob"))
				Expect(failed[0].Name).To(Equal(testCronjob))
				Expect(failed[0].Reason).To(ContainSubstring("strconv.ParseBool"))
			})
		})

		Context("with multiple cronjobs", func() {
			BeforeEach(func() {
				mockPeriod.Type = "down"

				// Create multiple test cronjobs
				cronjobNames := []string{"cronjob1", "cronjob2", "cronjob3"}
				for _, name := range cronjobNames {
					testCronJob := &batchV1.CronJob{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      name,
							Namespace: testNamespace,
						},
						Spec: batchV1.CronJobSpec{
							Suspend: ptr.To(false),
						},
					}

					_, err := fakeClient.BatchV1().CronJobs(testNamespace).Create(ctx, testCronJob, metaV1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should process all cronjobs", func() {
				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(3))
				Expect(failed).To(HaveLen(0))

				// Verify all cronjobs were suspended
				for _, name := range []string{"cronjob1", "cronjob2", "cronjob3"} {
					cronjob, err := fakeClient.BatchV1().CronJobs(testNamespace).Get(ctx, name, metaV1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(*cronjob.Spec.Suspend).To(Equal(true))
				}
			})

			It("should continue processing when one cronjob fails", func() {
				// Mock one cronjob to fail on update
				fakeClient.PrependReactor("update", "cronjobs", func(action testing.Action) (bool, runtime.Object, error) {
					updateAction := action.(testing.UpdateAction)
					if updateAction.GetObject().(*batchV1.CronJob).Name == "cronjob2" {
						return true, nil, errors.New("update error for cronjob2")
					}
					return false, nil, nil
				})

				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(2))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Name).To(Equal("cronjob2"))
			})
		})

		Context("with multiple namespaces", func() {
			var secondNamespace = "second-namespace"

			BeforeEach(func() {
				mockPeriod.Type = "down"

				// Add second namespace
				cronjobs.Resource.NsList = []string{testNamespace, secondNamespace}

				// Create cronjobs in both namespaces
				for _, ns := range []string{testNamespace, secondNamespace} {
					testCronJob := &batchV1.CronJob{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      fmt.Sprintf("cronjob-%s", ns),
							Namespace: ns,
						},
						Spec: batchV1.CronJobSpec{
							Suspend: ptr.To(false),
						},
					}

					_, err := fakeClient.BatchV1().CronJobs(ns).Create(ctx, testCronJob, metaV1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should process cronjobs from all namespaces", func() {
				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(2))
				Expect(failed).To(HaveLen(0))

				// Verify cronjobs in both namespaces were processed
				for _, ns := range []string{testNamespace, secondNamespace} {
					cronjob, err := fakeClient.BatchV1().CronJobs(ns).Get(ctx, fmt.Sprintf("cronjob-%s", ns), metaV1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(*cronjob.Spec.Suspend).To(Equal(true))
				}
			})
		})

		Context("edge cases", func() {
			It("should handle cronjob with nil suspend", func() {
				mockPeriod.Type = "down"

				// Create a cronjob with nil suspend (defaults to false)
				testCronJob := &batchV1.CronJob{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testCronjob,
						Namespace: testNamespace,
					},
					Spec: batchV1.CronJobSpec{
						Suspend: nil, // This will default to false
					},
				}

				_, err := fakeClient.BatchV1().CronJobs(testNamespace).Create(ctx, testCronJob, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))

				// Verify the cronjob was updated
				updatedCronJob, err := fakeClient.BatchV1().CronJobs(testNamespace).Get(ctx, testCronjob, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedCronJob.Spec.Suspend).To(Equal(true))
			})

			It("should handle empty namespace list", func() {
				cronjobs.Resource.NsList = []string{}

				success, failed, err := cronjobs.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(0))
			})
		})
	})
})
