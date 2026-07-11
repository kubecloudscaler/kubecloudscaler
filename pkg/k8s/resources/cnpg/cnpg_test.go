package cnpg_test

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

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/cnpg"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

const periodTypeRestore = "restore"

var (
	gvr = schema.GroupVersionResource{
		Group:    "postgresql.cnpg.io",
		Version:  "v1",
		Resource: "clusters",
	}
	clusterGVK = schema.GroupVersionKind{
		Group:   "postgresql.cnpg.io",
		Version: "v1",
		Kind:    "Cluster",
	}
)

func newCluster(name, namespace string, annotations map[string]string) *cnpg.Cluster {
	return &cnpg.Cluster{
		TypeMeta: metaV1.TypeMeta{
			Kind:       clusterGVK.Kind,
			APIVersion: fmt.Sprintf("%s/%s", clusterGVK.Group, clusterGVK.Version),
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
	}
}

var _ = Describe("Cnpg", func() {
	var (
		ctx        context.Context
		scheme     *runtime.Scheme
		dynClient  *dynamicfake.FakeDynamicClient
		resource   *utils.K8sResource
		mockPeriod *period.Period
		manager    *cnpg.Cnpg
	)

	setupManager := func(objs ...runtime.Object) {
		dynClient = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
			scheme,
			map[schema.GroupVersionResource]string{
				gvr: "ClusterList",
			},
			objs...,
		)

		manager = &cnpg.Cnpg{
			Resource:          resource,
			Logger:            &log.Logger,
			AnnotationManager: utils.NewAnnotationManager(),
		}
		manager.Client = dynClient.Resource(gvr)
	}

	getCluster := func(namespace, name string) *cnpg.Cluster {
		obj, err := dynClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metaV1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		cluster := &cnpg.Cluster{}
		Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, cluster)).To(Succeed())
		return cluster
	}

	origValueKey := utils.AnnotationsPrefix + "/" + utils.AnnotationsOrigValue

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(cnpg.AddToScheme(scheme)).To(Succeed())

		mockPeriod = &period.Period{
			Type:      common.PeriodTypeDown,
			IsActive:  true,
			StartTime: time.Now(),
			EndTime:   time.Now(),
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

	Context("scaling down (hibernate)", func() {
		It("hibernates the cluster and records the original state", func() {
			mockPeriod.Type = common.PeriodTypeDown

			setupManager(newCluster("cluster-down", "test-ns", nil))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(failed).To(BeEmpty())
			Expect(success).To(HaveLen(1))
			Expect(success[0].Kind).To(Equal("cnpgcluster"))
			Expect(success[0].Name).To(Equal("cluster-down"))

			updated := getCluster("test-ns", "cluster-down")
			Expect(updated.Annotations).To(HaveKeyWithValue(base.CNPGHibernationAnnotation, base.CNPGHibernationOn))
			Expect(updated.Annotations).To(HaveKeyWithValue(origValueKey, "false"))
		})

		It("preserves unrelated annotations when hibernating", func() {
			mockPeriod.Type = common.PeriodTypeDown

			setupManager(newCluster("cluster-keep", "test-ns", map[string]string{
				"team": "platform",
			}))

			success, _, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(HaveLen(1))

			updated := getCluster("test-ns", "cluster-keep")
			Expect(updated.Annotations).To(HaveKeyWithValue("team", "platform"))
			Expect(updated.Annotations).To(HaveKeyWithValue(base.CNPGHibernationAnnotation, base.CNPGHibernationOn))
		})
	})

	Context("scaling up (resume)", func() {
		It("resumes a hibernated cluster", func() {
			mockPeriod.Type = common.PeriodTypeUp

			setupManager(newCluster("cluster-up", "test-ns", map[string]string{
				base.CNPGHibernationAnnotation: base.CNPGHibernationOn,
			}))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(failed).To(BeEmpty())
			Expect(success).To(HaveLen(1))

			updated := getCluster("test-ns", "cluster-up")
			Expect(updated.Annotations).To(HaveKeyWithValue(base.CNPGHibernationAnnotation, base.CNPGHibernationOff))
			Expect(updated.Annotations).To(HaveKeyWithValue(origValueKey, "true"))
		})
	})

	Context("restoring (going out of period)", func() {
		It("restores a cluster that was hibernated before the scaler acted", func() {
			mockPeriod.Type = periodTypeRestore

			setupManager(newCluster("cluster-was-on", "test-ns", map[string]string{
				base.CNPGHibernationAnnotation: base.CNPGHibernationOff,
				origValueKey:                   "true",
			}))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(failed).To(BeEmpty())
			Expect(success).To(HaveLen(1))

			updated := getCluster("test-ns", "cluster-was-on")
			Expect(updated.Annotations).To(HaveKeyWithValue(base.CNPGHibernationAnnotation, base.CNPGHibernationOn))
			Expect(updated.Annotations).ToNot(HaveKey(origValueKey))
		})

		It("removes the hibernation annotation for a cluster that was not hibernated before", func() {
			mockPeriod.Type = periodTypeRestore

			setupManager(newCluster("cluster-was-off", "test-ns", map[string]string{
				base.CNPGHibernationAnnotation: base.CNPGHibernationOn,
				origValueKey:                   "false",
			}))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(failed).To(BeEmpty())
			Expect(success).To(HaveLen(1))

			updated := getCluster("test-ns", "cluster-was-off")
			Expect(updated.Annotations).ToNot(HaveKey(base.CNPGHibernationAnnotation))
			Expect(updated.Annotations).ToNot(HaveKey(origValueKey))
		})

		It("ignores clusters that were never touched by the scaler", func() {
			mockPeriod.Type = periodTypeRestore

			setupManager(newCluster("cluster-untouched", "test-ns", nil))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(BeEmpty())
			Expect(failed).To(BeEmpty())
		})
	})

	Context("error handling", func() {
		It("returns list error", func() {
			setupManager()

			dynClient.PrependReactor("list", "clusters", func(_ testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("boom")
			})

			success, failed, err := manager.SetState(ctx)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error listing cnpgclusters"))
			Expect(success).To(BeEmpty())
			Expect(failed).To(BeEmpty())
		})

		It("records restore failures when the recorded state is invalid", func() {
			mockPeriod.Type = periodTypeRestore

			setupManager(newCluster("cluster-bad", "test-ns", map[string]string{
				base.CNPGHibernationAnnotation: base.CNPGHibernationOn,
				origValueKey:                   "not-a-bool",
			}))

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(BeEmpty())
			Expect(failed).To(HaveLen(1))
			Expect(failed[0].Name).To(Equal("cluster-bad"))
			Expect(failed[0].Reason).To(ContainSubstring("error parsing bool value"))
		})

		It("records update failures", func() {
			setupManager(newCluster("cluster-update", "test-ns", nil))

			dynClient.PrependReactor("update", "clusters", func(_ testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("update failure")
			})

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(BeEmpty())
			Expect(failed).To(HaveLen(1))
			Expect(failed[0].Name).To(Equal("cluster-update"))
			Expect(failed[0].Reason).To(ContainSubstring("update failure"))
		})
	})

	Context("multi namespace management", func() {
		It("hibernates clusters across namespaces", func() {
			resource.NsList = []string{"ns-one", "ns-two"}
			mockPeriod.Type = common.PeriodTypeDown

			setupManager(
				newCluster("cluster-one", "ns-one", nil),
				newCluster("cluster-two", "ns-two", nil),
			)

			success, failed, err := manager.SetState(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(success).To(HaveLen(2))
			Expect(failed).To(BeEmpty())

			Expect(getCluster("ns-one", "cluster-one").Annotations).
				To(HaveKeyWithValue(base.CNPGHibernationAnnotation, base.CNPGHibernationOn))
			Expect(getCluster("ns-two", "cluster-two").Annotations).
				To(HaveKeyWithValue(base.CNPGHibernationAnnotation, base.CNPGHibernationOn))
		})
	})
})
