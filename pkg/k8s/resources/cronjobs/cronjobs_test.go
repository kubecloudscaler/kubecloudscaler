package cronjobs_test

import (
	"context"

	cloudscaleriov1alpha1 "github.com/cloudscalerio/cloudscaler/api/v1alpha1"
	"github.com/cloudscalerio/cloudscaler/pkg/k8s/resources/cronjobs"
	"github.com/cloudscalerio/cloudscaler/pkg/k8s/utils"
	periodPkg "github.com/cloudscalerio/cloudscaler/pkg/period"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var _ = Describe("Cronjobs", func() {
	var (
		cronjobsOk *cronjobs.Cronjobs
		cronjob1   *batchV1.CronJob
		cronjob2   *batchV1.CronJob
	)

	ctx := context.Background()

	BeforeEach(func() {
		cronjob1 = &batchV1.CronJob{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "cronjob1",
				Namespace: "ns1",
			},
			Spec: batchV1.CronJobSpec{
				Suspend: ptr.To(false),
			},
		}
		cronjob2 = &batchV1.CronJob{
			ObjectMeta: metaV1.ObjectMeta{
				Name:        "cronjob2",
				Namespace:   "ns2",
				Annotations: map[string]string{},
			},
			Spec: batchV1.CronJobSpec{
				Suspend: ptr.To(false),
			},
		}
		cronjobsOk = &cronjobs.Cronjobs{
			Client: k8sClient.BatchV1(),
			Resource: &utils.K8sResource{
				NsList: []string{"ns1", "ns2"},
				Period: &periodPkg.Period{
					Type:   "down",
					Period: &cloudscaleriov1alpha1.RecurringPeriod{},
				},
			},
		}
	})

	Describe("Cronjobs", func() {
		Context("in all namespaces", func() {
			It("should be suspended", func() {
				_, _ = k8sClient.BatchV1().CronJobs("ns1").Create(ctx, cronjob1, metaV1.CreateOptions{})
				_, _ = k8sClient.BatchV1().CronJobs("ns2").Create(ctx, cronjob2, metaV1.CreateOptions{})
				_, _, err := cronjobsOk.SetState(ctx)
				Expect(err).ToNot(HaveOccurred())

				cron1, _ := k8sClient.BatchV1().CronJobs("ns1").Get(ctx, "cronjob1", metaV1.GetOptions{})
				Expect(cron1.Spec.Suspend).To(Equal(ptr.To(true)))

				cron2, _ := k8sClient.BatchV1().CronJobs("ns2").Get(ctx, "cronjob2", metaV1.GetOptions{})
				Expect(cron2.Spec.Suspend).To(Equal(ptr.To(true)))
			})
		})
	})
})
