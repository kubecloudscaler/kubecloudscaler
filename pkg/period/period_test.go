package period_test

import (
	"fmt"

	cloudscaleriov1alpha1 "github.com/cloudscalerio/cloudscaler/api/v1alpha1"
	"github.com/cloudscalerio/cloudscaler/pkg/period"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

var _ = Describe("Cronjobs", func() {
	var (
		periodRecurringOk      *cloudscaleriov1alpha1.ScalerPeriod
		periodRecurringDayErr  *cloudscaleriov1alpha1.ScalerPeriod
		periodRecurringTimeErr *cloudscaleriov1alpha1.ScalerPeriod
		periodFixedOk          *cloudscaleriov1alpha1.ScalerPeriod
		periodFixedErr         *cloudscaleriov1alpha1.ScalerPeriod
	)

	BeforeEach(func() {
		periodFixedOk = &cloudscaleriov1alpha1.ScalerPeriod{
			Type: "restore",
			Time: cloudscaleriov1alpha1.TimePeriod{
				Fixed: &cloudscaleriov1alpha1.FixedPeriod{
					StartTime: "2024-10-10 08:12:00",
					EndTime:   "2024-10-11 08:12:00",
					Once:      ptr.To(false),
					// Timezone:  ptr.To("UTC"),
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
		periodFixedErr = &cloudscaleriov1alpha1.ScalerPeriod{
			Type: "restore",
			Time: cloudscaleriov1alpha1.TimePeriod{
				Fixed: &cloudscaleriov1alpha1.FixedPeriod{
					StartTime: "00:00",
					EndTime:   "00:00",
					Once:      ptr.To(false),
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
		periodRecurringOk = &cloudscaleriov1alpha1.ScalerPeriod{
			Type: "restore",
			Time: cloudscaleriov1alpha1.TimePeriod{
				Recurring: &cloudscaleriov1alpha1.RecurringPeriod{
					Days: []string{
						"all",
					},
					StartTime: "00:00",
					EndTime:   "01:00",
					Once:      ptr.To(false),
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
		periodRecurringDayErr = &cloudscaleriov1alpha1.ScalerPeriod{
			Type: "restore",
			Time: cloudscaleriov1alpha1.TimePeriod{
				Recurring: &cloudscaleriov1alpha1.RecurringPeriod{
					Days: []string{
						"test",
					},
					StartTime: "00:00",
					EndTime:   "00:12",
					Once:      ptr.To(false),
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
		periodRecurringTimeErr = &cloudscaleriov1alpha1.ScalerPeriod{
			Type: "restore",
			Time: cloudscaleriov1alpha1.TimePeriod{
				Recurring: &cloudscaleriov1alpha1.RecurringPeriod{
					Days: []string{
						"all",
					},
					StartTime: "2024-10-10 08:12:00",
					EndTime:   "2024-10-10 08:12:00",
					Once:      ptr.To(false),
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
	})

	Describe("Periods", func() {
		Context("Recurring", func() {
			It("should not error", func() {
				_, err := period.New(periodRecurringOk)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should error on day", func() {
				_, err := period.New(periodRecurringDayErr)
				Expect(err).To(Equal(fmt.Errorf("%w: %s", period.ErrBadDay, "test")))
			})
			It("should error on time", func() {
				_, err := period.New(periodRecurringTimeErr)
				Expect(err).To(Equal(period.ErrRecurringTimeFormat))
			})
		})
		Context("Fixed", func() {
			It("should not error", func() {
				_, err := period.New(periodFixedOk)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should error", func() {
				_, err := period.New(periodFixedErr)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
