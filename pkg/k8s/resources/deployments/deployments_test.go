package deployments_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	appsV1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	deploymentsPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/deployments"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

var _ = Describe("Deployments", func() {
	var (
		ctx           context.Context
		deployments   *deploymentsPkg.Deployments
		fakeClient    *fake.Clientset
		testNamespace = "test-namespace"
		testDeploy    = "test-deployment"
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

			deployments = &deploymentsPkg.Deployments{
				Resource: resource,
				Logger:   &log.Logger,
			}
			deployments.Client = fakeClient.AppsV1()
		})

		Context("when scaling down deployments", func() {
			BeforeEach(func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 0

				// Create a test deployment
				testDeployment := &appsV1.Deployment{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testDeploy,
						Namespace: testNamespace,
					},
					Spec: appsV1.DeploymentSpec{
						Replicas: ptr.To(int32(3)),
					},
				}

				// Add the deployment to the fake client
				_, err := fakeClient.AppsV1().Deployments(testNamespace).Create(ctx, testDeployment, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should scale down deployment successfully", func() {
				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("deployment"))
				Expect(success[0].Name).To(Equal(testDeploy))

				// Verify the deployment was updated
				updatedDeployment, err := fakeClient.AppsV1().Deployments(testNamespace).Get(ctx, testDeploy, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedDeployment.Spec.Replicas).To(Equal(int32(0)))
			})

			It("should add annotations when scaling down", func() {
				success, _, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Verify annotations were added
				updatedDeployment, err := fakeClient.AppsV1().Deployments(testNamespace).Get(ctx, testDeploy, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedDeployment.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsOrigValue))
			})
		})

		Context("when scaling up deployments", func() {
			BeforeEach(func() {
				mockPeriod.Type = "up"
				mockPeriod.MaxReplicas = 5

				// Create a test deployment
				testDeployment := &appsV1.Deployment{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testDeploy,
						Namespace: testNamespace,
					},
					Spec: appsV1.DeploymentSpec{
						Replicas: ptr.To(int32(1)),
					},
				}

				// Add the deployment to the fake client
				_, err := fakeClient.AppsV1().Deployments(testNamespace).Create(ctx, testDeployment, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should scale up deployment successfully", func() {
				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("deployment"))
				Expect(success[0].Name).To(Equal(testDeploy))

				// Verify the deployment was updated
				updatedDeployment, err := fakeClient.AppsV1().Deployments(testNamespace).Get(ctx, testDeploy, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedDeployment.Spec.Replicas).To(Equal(int32(5)))
			})

			It("should add annotations when scaling up", func() {
				success, _, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Verify annotations were added
				updatedDeployment, err := fakeClient.AppsV1().Deployments(testNamespace).Get(ctx, testDeploy, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(updatedDeployment.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsOrigValue))
			})
		})

		Context("when restoring deployments", func() {
			BeforeEach(func() {
				mockPeriod.Type = "restore"

				// Create a test deployment with annotations
				testDeployment := &appsV1.Deployment{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testDeploy,
						Namespace: testNamespace,
						Annotations: map[string]string{
							"kubecloudscaler.cloud/original-value": "3",
						},
					},
					Spec: appsV1.DeploymentSpec{
						Replicas: ptr.To(int32(1)),
					},
				}

				// Add the deployment to the fake client
				_, err := fakeClient.AppsV1().Deployments(testNamespace).Create(ctx, testDeployment, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should restore deployment successfully", func() {
				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))
				Expect(success[0].Kind).To(Equal("deployment"))
				Expect(success[0].Name).To(Equal(testDeploy))

				// Verify the deployment was restored
				updatedDeployment, err := fakeClient.AppsV1().Deployments(testNamespace).Get(ctx, testDeploy, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedDeployment.Spec.Replicas).To(Equal(int32(3)))
			})

			It("should handle already restored deployments", func() {
				// First restore
				success, _, err := deployments.SetState(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))

				// Second restore should not change anything since annotations are removed
				success, _, err = deployments.SetState(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0)) // No action needed
			})
		})

		Context("when handling errors", func() {
			It("should handle deployment list error", func() {
				// Mock the fake client to return an error when listing deployments
				fakeClient.PrependReactor("list", "deployments", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("list error")
				})

				success, failed, err := deployments.SetState(ctx)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error listing deployments"))
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(0))
			})

			It("should handle deployment get error", func() {
				// Create a deployment in the fake client
				testDeployment := &appsV1.Deployment{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testDeploy,
						Namespace: testNamespace,
					},
					Spec: appsV1.DeploymentSpec{
						Replicas: ptr.To(int32(3)),
					},
				}

				_, err := fakeClient.AppsV1().Deployments(testNamespace).Create(ctx, testDeployment, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Mock the fake client to return an error when getting a specific deployment
				fakeClient.PrependReactor("get", "deployments", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("get error")
				})

				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("deployment"))
				Expect(failed[0].Name).To(Equal(testDeploy))
				Expect(failed[0].Reason).To(ContainSubstring("get error"))
			})

			It("should handle deployment update error", func() {
				// Create a deployment in the fake client
				testDeployment := &appsV1.Deployment{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testDeploy,
						Namespace: testNamespace,
					},
					Spec: appsV1.DeploymentSpec{
						Replicas: ptr.To(int32(3)),
					},
				}

				_, err := fakeClient.AppsV1().Deployments(testNamespace).Create(ctx, testDeployment, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Mock the fake client to return an error when updating deployments
				fakeClient.PrependReactor("update", "deployments", func(action testing.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("update error")
				})

				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("deployment"))
				Expect(failed[0].Name).To(Equal(testDeploy))
				Expect(failed[0].Reason).To(ContainSubstring("update error"))
			})

			It("should handle annotation restoration error", func() {
				mockPeriod.Type = "restore"

				// Create a deployment with invalid annotations
				testDeployment := &appsV1.Deployment{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testDeploy,
						Namespace: testNamespace,
						Annotations: map[string]string{
							"kubecloudscaler.cloud/original-value": "invalid",
						},
					},
					Spec: appsV1.DeploymentSpec{
						Replicas: ptr.To(int32(1)),
					},
				}

				_, err := fakeClient.AppsV1().Deployments(testNamespace).Create(ctx, testDeployment, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Kind).To(Equal("deployment"))
				Expect(failed[0].Name).To(Equal(testDeploy))
				Expect(failed[0].Reason).To(ContainSubstring("strconv.Atoi"))
			})
		})

		Context("with multiple deployments", func() {
			BeforeEach(func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1

				// Create multiple test deployments
				deploymentNames := []string{"deploy1", "deploy2", "deploy3"}
				for _, name := range deploymentNames {
					testDeployment := &appsV1.Deployment{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      name,
							Namespace: testNamespace,
						},
						Spec: appsV1.DeploymentSpec{
							Replicas: ptr.To(int32(3)),
						},
					}

					_, err := fakeClient.AppsV1().Deployments(testNamespace).Create(ctx, testDeployment, metaV1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should process all deployments", func() {
				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(3))
				Expect(failed).To(HaveLen(0))

				// Verify all deployments were scaled down
				for _, name := range []string{"deploy1", "deploy2", "deploy3"} {
					deployment, err := fakeClient.AppsV1().Deployments(testNamespace).Get(ctx, name, metaV1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(*deployment.Spec.Replicas).To(Equal(int32(1)))
				}
			})

			It("should continue processing when one deployment fails", func() {
				// Mock one deployment to fail on update
				fakeClient.PrependReactor("update", "deployments", func(action testing.Action) (bool, runtime.Object, error) {
					updateAction := action.(testing.UpdateAction)
					if updateAction.GetObject().(*appsV1.Deployment).Name == "deploy2" {
						return true, nil, errors.New("update error for deploy2")
					}
					return false, nil, nil
				})

				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(2))
				Expect(failed).To(HaveLen(1))
				Expect(failed[0].Name).To(Equal("deploy2"))
			})
		})

		Context("with multiple namespaces", func() {
			var secondNamespace = "second-namespace"

			BeforeEach(func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1

				// Add second namespace
				deployments.Resource.NsList = []string{testNamespace, secondNamespace}

				// Create deployments in both namespaces
				for _, ns := range []string{testNamespace, secondNamespace} {
					testDeployment := &appsV1.Deployment{
						ObjectMeta: metaV1.ObjectMeta{
							Name:      fmt.Sprintf("deploy-%s", ns),
							Namespace: ns,
						},
						Spec: appsV1.DeploymentSpec{
							Replicas: ptr.To(int32(3)),
						},
					}

					_, err := fakeClient.AppsV1().Deployments(ns).Create(ctx, testDeployment, metaV1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should process deployments from all namespaces", func() {
				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(2))
				Expect(failed).To(HaveLen(0))

				// Verify deployments in both namespaces were processed
				for _, ns := range []string{testNamespace, secondNamespace} {
					deployment, err := fakeClient.AppsV1().Deployments(ns).Get(ctx, fmt.Sprintf("deploy-%s", ns), metaV1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(*deployment.Spec.Replicas).To(Equal(int32(1)))
				}
			})
		})

		Context("edge cases", func() {
			It("should handle deployment with nil replicas", func() {
				mockPeriod.Type = "down"
				mockPeriod.MinReplicas = 1

				// Create a deployment with nil replicas (defaults to 1)
				testDeployment := &appsV1.Deployment{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testDeploy,
						Namespace: testNamespace,
					},
					Spec: appsV1.DeploymentSpec{
						Replicas: nil, // This will default to 1
					},
				}

				_, err := fakeClient.AppsV1().Deployments(testNamespace).Create(ctx, testDeployment, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))

				// Verify the deployment was updated
				updatedDeployment, err := fakeClient.AppsV1().Deployments(testNamespace).Get(ctx, testDeploy, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedDeployment.Spec.Replicas).To(Equal(int32(1)))
			})

			It("should handle deployment with zero replicas", func() {
				mockPeriod.Type = "up"
				mockPeriod.MaxReplicas = 5

				// Create a deployment with zero replicas
				testDeployment := &appsV1.Deployment{
					ObjectMeta: metaV1.ObjectMeta{
						Name:      testDeploy,
						Namespace: testNamespace,
					},
					Spec: appsV1.DeploymentSpec{
						Replicas: ptr.To(int32(0)),
					},
				}

				_, err := fakeClient.AppsV1().Deployments(testNamespace).Create(ctx, testDeployment, metaV1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(1))
				Expect(failed).To(HaveLen(0))

				// Verify the deployment was updated
				updatedDeployment, err := fakeClient.AppsV1().Deployments(testNamespace).Get(ctx, testDeploy, metaV1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(*updatedDeployment.Spec.Replicas).To(Equal(int32(5)))
			})

			It("should handle empty namespace list", func() {
				deployments.Resource.NsList = []string{}

				success, failed, err := deployments.SetState(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(success).To(HaveLen(0))
				Expect(failed).To(HaveLen(0))
			})
		})
	})
})
