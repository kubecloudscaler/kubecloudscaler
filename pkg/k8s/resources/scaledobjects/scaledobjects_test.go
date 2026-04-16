package scaledobjects_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog/log"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/testing"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/scaledobjects"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

var (
	gvr = schema.GroupVersionResource{
		Group:    "keda.sh",
		Version:  "v1alpha1",
		Resource: "scaledobjects",
	}
	scaledObjectGVK = schema.GroupVersionKind{
		Group:   "keda.sh",
		Version: "v1alpha1",
		Kind:    "ScaledObject",
	}
)

func newScaledObject(name, namespace string, minReplicas, maxReplicas int32, annotations map[string]string) *scaledobjects.ScaledObject {
	return &scaledobjects.ScaledObject{
		TypeMeta: metaV1.TypeMeta{
			Kind:       scaledObjectGVK.Kind,
			APIVersion: fmt.Sprintf("%s/%s", scaledObjectGVK.Group, scaledObjectGVK.Version),
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Spec: scaledobjects.ScaledObjectSpec{
			MinReplicaCount: ptr.To(minReplicas),
			MaxReplicaCount: ptr.To(maxReplicas),
		},
	}
}

var _ = Describe("ScaledObjects", func() {
	var (
		ctx        context.Context
		scheme     *runtime.Scheme
		dynClient  *dynamicfake.FakeDynamicClient
		resource   *utils.K8sResource
		mockPeriod *period.Period
		manager    *scaledobjects.ScaledObjects
	)

	setupManager := func(objs ...runtime.Object) {
		dynClient = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
			scheme,
			map[schema.GroupVersionResource]string{
				gvr: "ScaledObjectList",
			},
			objs...,
		)

		manager = &scaledobjects.ScaledObjects{
			Resource:          resource,
			Logger:            &log.Logger,
			AnnotationManager: utils.NewAnnotationManager(),
		}
		manager.Client = dynClient.Resource(gvr)
	}

	getScaledObject := func(namespace, name string) *scaledobjects.ScaledObject {
		obj, err := dynClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metaV1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		so := &scaledobjects.ScaledObject{}
		Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, so)).To(Succeed())
		return so
	}

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(scaledobjects.AddToScheme(scheme)).To(Succeed())

		mockPeriod = &period.Period{
			Type:        common.PeriodTypeDown,
			MinReplicas: 0,
			MaxReplicas: 5,
			IsActive:    true,
			StartTime:   time.Now(),
			EndTime:     time.Now(),
			Spec: &common.RecurringPeriod{
				Days:      []common.DayOfWeek{common.DayAll},
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

	Context("scaling down with minReplicas=0 (KEDA pause)", func() {
		It("adds KEDA pause annotations when period is down and minReplicas is 0", func() {
			mockPeriod.Type = common.PeriodTypeDown
			mockPeriod.MinReplicas = 0

			setupManager(newScaledObject("so-pause", "test-ns", 2, 10, nil))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(failed).To(BeEmpty())
			Expect(success).To(HaveLen(1))
			Expect(success[0].Kind).To(Equal("scaledobject"))
			Expect(success[0].Name).To(Equal("so-pause"))

			updated := getScaledObject("test-ns", "so-pause")
			Expect(updated.Annotations).To(HaveKeyWithValue(base.KedaPausedAnnotation, "true"))
			Expect(updated.Annotations).To(HaveKeyWithValue(base.KedaPausedReplicasAnnotation, "0"))
			Expect(updated.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue))
			Expect(updated.Annotations).To(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMaxOrigValue))
		})
	})

	Context("scaling down with minReplicas > 0 (standard min/max)", func() {
		It("sets min/max replicas without KEDA pause annotations", func() {
			mockPeriod.Type = common.PeriodTypeDown
			mockPeriod.MinReplicas = 2
			mockPeriod.MaxReplicas = 4

			setupManager(newScaledObject("so-minmax", "test-ns", 5, 10, nil))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(failed).To(BeEmpty())
			Expect(success).To(HaveLen(1))

			updated := getScaledObject("test-ns", "so-minmax")
			Expect(updated.Annotations).ToNot(HaveKey(base.KedaPausedAnnotation))
			Expect(updated.Annotations).ToNot(HaveKey(base.KedaPausedReplicasAnnotation))
			Expect(*updated.Spec.MinReplicaCount).To(Equal(int32(2)))
			Expect(*updated.Spec.MaxReplicaCount).To(Equal(int32(4)))
		})
	})

	Context("scaling up", func() {
		It("scales up with min/max replicas", func() {
			mockPeriod.Type = common.PeriodTypeUp
			mockPeriod.MinReplicas = 8
			mockPeriod.MaxReplicas = 12

			setupManager(newScaledObject("so-up", "test-ns", 2, 5, nil))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(HaveLen(1))
			Expect(failed).To(BeEmpty())

			updated := getScaledObject("test-ns", "so-up")
			Expect(*updated.Spec.MinReplicaCount).To(Equal(int32(8)))
			Expect(*updated.Spec.MaxReplicaCount).To(Equal(int32(12)))
			Expect(updated.Annotations).To(HaveKeyWithValue(utils.AnnotationsPrefix+"/"+utils.AnnotationsMinOrigValue, "2"))
			Expect(updated.Annotations).To(HaveKeyWithValue(utils.AnnotationsPrefix+"/"+utils.AnnotationsMaxOrigValue, "5"))
		})

		It("removes KEDA pause annotations when transitioning directly from a paused down period", func() {
			mockPeriod.Type = common.PeriodTypeUp
			mockPeriod.MinReplicas = 3
			mockPeriod.MaxReplicas = 8

			so := newScaledObject("so-unpause", "test-ns", 0, 0, map[string]string{
				utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue: "3",
				utils.AnnotationsPrefix + "/" + utils.AnnotationsMaxOrigValue: "8",
				base.KedaPausedAnnotation:                                     "true",
				base.KedaPausedReplicasAnnotation:                             "0",
			})

			setupManager(so)

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(failed).To(BeEmpty())
			Expect(success).To(HaveLen(1))

			updated := getScaledObject("test-ns", "so-unpause")
			Expect(updated.Annotations).ToNot(HaveKey(base.KedaPausedAnnotation))
			Expect(updated.Annotations).ToNot(HaveKey(base.KedaPausedReplicasAnnotation))
			Expect(*updated.Spec.MinReplicaCount).To(Equal(int32(3)))
			Expect(*updated.Spec.MaxReplicaCount).To(Equal(int32(8)))
		})
	})

	Context("restoring (going out of period)", func() {
		It("restores original values and removes KEDA pause annotations", func() {
			mockPeriod.Type = "restore"

			so := newScaledObject("so-restore", "test-ns", 0, 0, map[string]string{
				utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue: "3",
				utils.AnnotationsPrefix + "/" + utils.AnnotationsMaxOrigValue: "10",
				base.KedaPausedAnnotation:                                     "true",
				base.KedaPausedReplicasAnnotation:                             "0",
			})

			setupManager(so)

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(HaveLen(1))
			Expect(failed).To(BeEmpty())

			updated := getScaledObject("test-ns", "so-restore")
			Expect(updated.Annotations).ToNot(HaveKey(base.KedaPausedAnnotation))
			Expect(updated.Annotations).ToNot(HaveKey(base.KedaPausedReplicasAnnotation))
			Expect(updated.Annotations).ToNot(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue))
			Expect(updated.Annotations).ToNot(HaveKey(utils.AnnotationsPrefix + "/" + utils.AnnotationsMaxOrigValue))
			Expect(*updated.Spec.MinReplicaCount).To(Equal(int32(3)))
			Expect(*updated.Spec.MaxReplicaCount).To(Equal(int32(10)))
		})

		It("restores original values when no KEDA pause annotations present", func() {
			mockPeriod.Type = "restore"

			so := newScaledObject("so-restore-nopause", "test-ns", 2, 4, map[string]string{
				utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue: "5",
				utils.AnnotationsPrefix + "/" + utils.AnnotationsMaxOrigValue: "15",
			})

			setupManager(so)

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(HaveLen(1))
			Expect(failed).To(BeEmpty())

			updated := getScaledObject("test-ns", "so-restore-nopause")
			Expect(*updated.Spec.MinReplicaCount).To(Equal(int32(5)))
			Expect(*updated.Spec.MaxReplicaCount).To(Equal(int32(15)))
		})

		It("ignores already restored scaled objects", func() {
			mockPeriod.Type = "restore"

			setupManager(newScaledObject("so-already", "test-ns", 2, 5, nil))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(BeEmpty())
			Expect(failed).To(BeEmpty())
		})
	})

	Context("error handling", func() {
		It("returns list error", func() {
			setupManager()

			dynClient.Fake.PrependReactor("list", "scaledobjects", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("boom")
			})

			success, failed, err := manager.SetState(ctx)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error listing scaledobjects"))
			Expect(success).To(BeEmpty())
			Expect(failed).To(BeEmpty())
		})

		It("returns validation error when annotations invalid", func() {
			mockPeriod.Type = "restore"

			so := newScaledObject("so-bad", "test-ns", 1, 2, map[string]string{
				utils.AnnotationsPrefix + "/" + utils.AnnotationsMinOrigValue: "invalid",
			})

			setupManager(so)

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(BeEmpty())
			Expect(failed).To(HaveLen(1))
			Expect(failed[0].Name).To(Equal("so-bad"))
			Expect(failed[0].Reason).To(ContainSubstring("error parsing min value"))
		})

		It("records update failures", func() {
			setupManager(newScaledObject("so-update", "test-ns", 2, 3, nil))

			dynClient.Fake.PrependReactor("update", "scaledobjects", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("update failure")
			})

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(BeEmpty())
			Expect(failed).To(HaveLen(1))
			Expect(failed[0].Name).To(Equal("so-update"))
			Expect(failed[0].Reason).To(ContainSubstring("update failure"))
		})
	})

	Context("multi namespace management", func() {
		It("processes scaled objects across namespaces", func() {
			resource.NsList = []string{"ns-one", "ns-two"}
			mockPeriod.Type = common.PeriodTypeDown
			mockPeriod.MinReplicas = 0

			setupManager(
				newScaledObject("so-one", "ns-one", 5, 6, nil),
				newScaledObject("so-two", "ns-two", 6, 7, nil),
			)

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(HaveLen(2))
			Expect(failed).To(BeEmpty())

			// Both should have KEDA pause annotations
			for _, ns := range []string{"ns-one", "ns-two"} {
				updated := getScaledObject(ns, "so-"+ns[3:])
				Expect(updated.Annotations).To(HaveKeyWithValue(base.KedaPausedAnnotation, "true"))
				Expect(updated.Annotations).To(HaveKeyWithValue(base.KedaPausedReplicasAnnotation, "0"))
			}
		})
	})
})
