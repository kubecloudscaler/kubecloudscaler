package period_test

import (
	"fmt"

	"github.com/cloudscalerio/cloudscaler/api/common"
	"github.com/cloudscalerio/cloudscaler/pkg/period"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

var _ = Describe("Cronjobs", func() {
	var (
		periodRecurringOk      *common.ScalerPeriod
		periodRecurringDayErr  *common.ScalerPeriod
		periodRecurringTimeErr *common.ScalerPeriod
		periodFixedOk          *common.ScalerPeriod
		periodFixedErr         *common.ScalerPeriod
	)

	BeforeEach(func() {
		periodFixedOk = &common.ScalerPeriod{
			Type: "restore",
			Time: common.TimePeriod{
				Fixed: &common.FixedPeriod{
					StartTime: "2024-10-10 08:12:00",
					EndTime:   "2024-10-11 08:12:00",
					Once:      false,
					// Timezone:  ptr.To("UTC"),
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
		periodFixedErr = &common.ScalerPeriod{
			Type: "restore",
			Time: common.TimePeriod{
				Fixed: &common.FixedPeriod{
					StartTime: "00:00",
					EndTime:   "00:00",
					Once:      false,
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
		periodRecurringOk = &common.ScalerPeriod{
			Type: "restore",
			Time: common.TimePeriod{
				Recurring: &common.RecurringPeriod{
					Days: []string{
						"all",
					},
					StartTime: "00:00",
					EndTime:   "00:00",
					Once:      false,
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
		periodRecurringDayErr = &common.ScalerPeriod{
			Type: "restore",
			Time: common.TimePeriod{
				Recurring: &common.RecurringPeriod{
					Days: []string{
						"test",
					},
					StartTime: "00:00",
					EndTime:   "00:12",
					Once:      false,
				},
			},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(1)),
		}
		periodRecurringTimeErr = &common.ScalerPeriod{
			Type: "restore",
			Time: common.TimePeriod{
				Recurring: &common.RecurringPeriod{
					Days: []string{
						"all",
					},
					StartTime: "2024-10-10 08:12:00",
					EndTime:   "2024-10-10 08:12:00",
					Once:      false,
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
