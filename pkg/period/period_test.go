package period_test

import (
	"fmt"

	kubecloudscalerv1alpha1 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha1"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

var _ = Describe("Cronjobs", func() {
	var (
		periodRecurringOk      *kubecloudscalerv1alpha1.ScalerPeriod
		periodRecurringDayErr  *kubecloudscalerv1alpha1.ScalerPeriod
		periodRecurringTimeErr *kubecloudscalerv1alpha1.ScalerPeriod
		periodFixedOk          *kubecloudscalerv1alpha1.ScalerPeriod
		periodFixedErr         *kubecloudscalerv1alpha1.ScalerPeriod
	)

	BeforeEach(func() {
		periodFixedOk = &kubecloudscalerv1alpha1.ScalerPeriod{
			Type: "restore",
			Time: kubecloudscalerv1alpha1.TimePeriod{
				Fixed: &kubecloudscalerv1alpha1.FixedPeriod{
					StartTime: "2024-10-10 08:12:00",
					EndTime:   "2024-10-11 08:12:00",
					Once:      ptr.To(false),
					// Timezone:  ptr.To("UTC"),
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
		periodFixedErr = &kubecloudscalerv1alpha1.ScalerPeriod{
			Type: "restore",
			Time: kubecloudscalerv1alpha1.TimePeriod{
				Fixed: &kubecloudscalerv1alpha1.FixedPeriod{
					StartTime: "00:00",
					EndTime:   "00:00",
					Once:      ptr.To(false),
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
		periodRecurringOk = &kubecloudscalerv1alpha1.ScalerPeriod{
			Type: "restore",
			Time: kubecloudscalerv1alpha1.TimePeriod{
				Recurring: &kubecloudscalerv1alpha1.RecurringPeriod{
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
		periodRecurringDayErr = &kubecloudscalerv1alpha1.ScalerPeriod{
			Type: "restore",
			Time: kubecloudscalerv1alpha1.TimePeriod{
				Recurring: &kubecloudscalerv1alpha1.RecurringPeriod{
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
		periodRecurringTimeErr = &kubecloudscalerv1alpha1.ScalerPeriod{
			Type: "restore",
			Time: kubecloudscalerv1alpha1.TimePeriod{
				Recurring: &kubecloudscalerv1alpha1.RecurringPeriod{
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
