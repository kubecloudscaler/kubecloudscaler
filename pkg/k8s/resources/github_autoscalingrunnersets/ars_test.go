package ars_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	actionsV1alpha1 "github.com/actions/actions-runner-controller/apis/actions.github.com/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/testing"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	ars "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/github_autoscalingrunnersets"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

var (
	gvr = schema.GroupVersionResource{
		Group:    "actions.github.com",
		Version:  "v1alpha1",
		Resource: "autoscalingrunnersets",
	}
	runnerSetGVK = schema.GroupVersionKind{
		Group:   "actions.github.com",
		Version: "v1alpha1",
		Kind:    "AutoscalingRunnerSet",
	}
)

var _ = Describe("GithubAutoscalingRunnersets", func() {
	var (
		ctx        context.Context
		scheme     *runtime.Scheme
		dynClient  *dynamicfake.FakeDynamicClient
		resource   *utils.K8sResource
		mockPeriod *period.Period
		manager    *ars.GithubAutoscalingRunnersets
	)

	newRunnerSet := func(name, namespace string, min, max int) *actionsV1alpha1.AutoscalingRunnerSet {
		return &actionsV1alpha1.AutoscalingRunnerSet{
			TypeMeta: metaV1.TypeMeta{
				Kind:       runnerSetGVK.Kind,
				APIVersion: fmt.Sprintf("%s/%s", runnerSetGVK.Group, runnerSetGVK.Version),
			},
			ObjectMeta: metaV1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: actionsV1alpha1.AutoscalingRunnerSetSpec{
				Template:   corev1.PodTemplateSpec{},
				MinRunners: ptr.To(min),
				MaxRunners: ptr.To(max),
			},
		}
	}

	setupManager := func(objs ...runtime.Object) {
		dynClient = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
			scheme,
			map[schema.GroupVersionResource]string{
				gvr: "AutoscalingRunnerSetList",
			},
			objs...,
		)

		manager = &ars.GithubAutoscalingRunnersets{
			Resource: resource,
			Logger:   &log.Logger,
		}
		manager.Client = dynClient.Resource(gvr)
	}

	getRunnerSet := func(namespace, name string) *actionsV1alpha1.AutoscalingRunnerSet {
		obj, err := dynClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metaV1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		var runnerSet actionsV1alpha1.AutoscalingRunnerSet
		Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &runnerSet)).To(Succeed())

		return &runnerSet
	}

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(actionsV1alpha1.AddToScheme(scheme)).To(Succeed())

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
			NsList:      []string{"test-ns"},
			ListOptions: metaV1.ListOptions{},
			Period:      mockPeriod,
		}
	})

	Context("scaling operations", func() {
		It("scales down runner sets", func() {
			mockPeriod.Type = "down"
			mockPeriod.MinReplicas = 2
			mockPeriod.MaxReplicas = 4

			setupManager(newRunnerSet("rs-down", "test-ns", 5, 10))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(failed).To(BeEmpty())
			Expect(success).To(HaveLen(1))
			Expect(success[0].Kind).To(Equal("autoscalingrunnerset"))
			Expect(success[0].Name).To(Equal("rs-down"))

			updated := getRunnerSet("test-ns", "rs-down")
			Expect(*updated.Spec.MinRunners).To(Equal(2))
			Expect(*updated.Spec.MaxRunners).To(Equal(4))
			Expect(updated.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue))
			Expect(updated.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMaxOrigValue))
		})

		It("scales up runner sets", func() {
			mockPeriod.Type = "up"
			mockPeriod.MinReplicas = 8
			mockPeriod.MaxReplicas = 12

			setupManager(newRunnerSet("rs-up", "test-ns", 2, 5))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(HaveLen(1))
			Expect(failed).To(BeEmpty())

			updated := getRunnerSet("test-ns", "rs-up")
			Expect(*updated.Spec.MinRunners).To(Equal(8))
			Expect(*updated.Spec.MaxRunners).To(Equal(12))
		})

		It("restores original runners", func() {
			mockPeriod.Type = "restore"

			rs := newRunnerSet("rs-restore", "test-ns", 2, 5)
			rs.Annotations = map[string]string{
				utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue: "6",
				utils.AnnotationsPrefix + "/" + utils.AnnotationsMaxOrigValue: "15",
			}

			setupManager(rs)

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(HaveLen(1))
			Expect(failed).To(BeEmpty())

			updated := getRunnerSet("test-ns", "rs-restore")
			Expect(*updated.Spec.MinRunners).To(Equal(6))
			Expect(*updated.Spec.MaxRunners).To(Equal(15))
			Expect(updated.Annotations).ToNot(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue))
			Expect(updated.Annotations).ToNot(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMaxOrigValue))
		})

		It("ignores already restored runner sets", func() {
			mockPeriod.Type = "restore"

			setupManager(newRunnerSet("rs-restored", "test-ns", 2, 5))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(HaveLen(0))
			Expect(failed).To(HaveLen(0))
		})
	})

	Context("error handling", func() {
		It("returns list error", func() {
			setupManager()

			dynClient.Fake.PrependReactor("list", "autoscalingrunnersets", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("boom")
			})

			success, failed, err := manager.SetState(ctx)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error listing autoscaling runner sets"))
			Expect(success).To(BeEmpty())
			Expect(failed).To(BeEmpty())
		})

		It("returns validation error when annotations invalid", func() {
			mockPeriod.Type = "restore"

			rs := newRunnerSet("rs-bad", "test-ns", 1, 2)
			rs.Annotations = map[string]string{
				utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue: "invalid",
			}

			setupManager(rs)

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(BeEmpty())
			Expect(failed).To(HaveLen(1))
			Expect(failed[0].Name).To(Equal("rs-bad"))
			Expect(failed[0].Reason).To(ContainSubstring("error parsing min value"))
		})

		It("records update failures", func() {
			setupManager(newRunnerSet("rs-update", "test-ns", 2, 3))

			dynClient.Fake.PrependReactor("update", "autoscalingrunnersets", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("update failure")
			})

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(BeEmpty())
			Expect(failed).To(HaveLen(1))
			Expect(failed[0].Name).To(Equal("rs-update"))
			Expect(failed[0].Reason).To(ContainSubstring("update failure"))
		})
	})

	Context("multi namespace management", func() {
		It("processes runner sets across namespaces", func() {
			resource.NsList = []string{"ns-one", "ns-two"}
			mockPeriod.Type = "down"
			mockPeriod.MinReplicas = 1
			mockPeriod.MaxReplicas = 2

			setupManager(
				newRunnerSet("rs-one", "ns-one", 5, 6),
				newRunnerSet("rs-two", "ns-two", 6, 7),
			)

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(HaveLen(2))
			Expect(failed).To(BeEmpty())

			Expect(*getRunnerSet("ns-one", "rs-one").Spec.MinRunners).To(Equal(1))
			Expect(*getRunnerSet("ns-one", "rs-one").Spec.MaxRunners).To(Equal(2))
			Expect(*getRunnerSet("ns-two", "rs-two").Spec.MinRunners).To(Equal(1))
			Expect(*getRunnerSet("ns-two", "rs-two").Spec.MaxRunners).To(Equal(2))
		})
	})
})
