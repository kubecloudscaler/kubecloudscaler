package period_test

import (
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

var _ = Describe("Period", func() {

	Describe("New", func() {
		Context("with valid recurring periods", func() {
			var (
				periodRecurring *common.ScalerPeriod
			)

			BeforeEach(func() {
				periodRecurring = &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days: []string{
								"all",
							},
							StartTime:   "08:00",
							EndTime:     "18:00",
							Once:        ptr.To(false),
							GracePeriod: ptr.To("5m"),
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}
			})

			It("should create period successfully", func() {
				result, err := period.New(periodRecurring)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.Type).To(Equal("down"))
				Expect(result.MinReplicas).To(Equal(int32(1)))
				Expect(result.MaxReplicas).To(Equal(int32(5)))
				Expect(result.Period).ToNot(BeNil())
				Expect(result.Hash).ToNot(BeEmpty())
				Expect(result.GracePeriod).To(Equal(5 * time.Minute))
			})

			It("should handle specific days", func() {
				periodRecurring.Time.Recurring.Days = []string{"mon", "wed", "fri"}
				result, err := period.New(periodRecurring)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.Period.Days).To(Equal([]string{"mon", "wed", "fri"}))
			})

			It("should handle single day", func() {
				periodRecurring.Time.Recurring.Days = []string{"tue"}
				result, err := period.New(periodRecurring)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.Period.Days).To(Equal([]string{"tue"}))
			})

			It("should handle timezone", func() {
				periodRecurring.Time.Recurring.Timezone = ptr.To("America/New_York")
				result, err := period.New(periodRecurring)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.Period.Timezone).To(Equal(ptr.To("America/New_York")))
			})

			It("should handle reverse logic", func() {
				periodRecurring.Time.Recurring.Reverse = ptr.To(true)
				result, err := period.New(periodRecurring)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(*result.Period.Reverse).To(BeTrue())
			})

			It("should handle once flag", func() {
				periodRecurring.Time.Recurring.Once = ptr.To(true)
				result, err := period.New(periodRecurring)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(*result.Period.Once).To(BeTrue())
			})
		})

		Context("with valid fixed periods", func() {
			var (
				periodFixed *common.ScalerPeriod
			)

			BeforeEach(func() {
				periodFixed = &common.ScalerPeriod{
					Type: "up",
					Time: common.TimePeriod{
						Fixed: &common.FixedPeriod{
							StartTime:   "2024-10-10 08:00:00",
							EndTime:     "2024-10-10 18:00:00",
							Once:        ptr.To(true),
							GracePeriod: ptr.To("10m"),
						},
					},
					MinReplicas: ptr.To(int32(3)),
					MaxReplicas: ptr.To(int32(10)),
				}
			})

			It("should create period successfully", func() {
				result, err := period.New(periodFixed)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.Type).To(Equal("up"))
				Expect(result.MinReplicas).To(Equal(int32(3)))
				Expect(result.MaxReplicas).To(Equal(int32(10)))
				Expect(result.Period).ToNot(BeNil())
				Expect(result.Hash).ToNot(BeEmpty())
				Expect(result.GracePeriod).To(Equal(10 * time.Minute))
			})

			It("should convert fixed to recurring format", func() {
				result, err := period.New(periodFixed)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Period.Days).To(ContainElement("all"))
				Expect(result.Period.StartTime).To(Equal("2024-10-10 08:00:00"))
				Expect(result.Period.EndTime).To(Equal("2024-10-10 18:00:00"))
			})

			It("should handle timezone", func() {
				periodFixed.Time.Fixed.Timezone = ptr.To("Europe/London")
				result, err := period.New(periodFixed)

				Expect(err).ToNot(HaveOccurred())
				Expect(result).ToNot(BeNil())
				Expect(result.Period.Timezone).To(Equal(ptr.To("Europe/London")))
			})
		})

		Context("with invalid configurations", func() {
			It("should error on minReplicas greater than maxReplicas", func() {
				invalidPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(10)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(invalidPeriod)

				Expect(err).To(Equal(period.ErrMinReplicasGreaterThanMax))
				Expect(result).To(BeNil())
			})

			It("should error on invalid day notation", func() {
				invalidPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"invalid"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(invalidPeriod)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid day notation"))
				Expect(result).To(BeNil())
			})

			It("should error on invalid recurring time format", func() {
				invalidPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "invalid-time",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(invalidPeriod)

				Expect(err).To(Equal(period.ErrRecurringTimeFormat))
				Expect(result).To(BeNil())
			})

			It("should error on invalid fixed time format", func() {
				invalidPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Fixed: &common.FixedPeriod{
							StartTime: "invalid-datetime",
							EndTime:   "2024-10-10 18:00:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(invalidPeriod)

				Expect(err).To(Equal(period.ErrFixedTimeFormat))
				Expect(result).To(BeNil())
			})

			It("should error on start time after end time", func() {
				invalidPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "18:00",
							EndTime:   "08:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(invalidPeriod)

				Expect(err).To(Equal(period.ErrStartAfterEnd))
				Expect(result).To(BeNil())
			})

			It("should error on invalid timezone", func() {
				invalidPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "08:00",
							EndTime:   "18:00",
							Timezone:  ptr.To("Invalid/Timezone"),
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(invalidPeriod)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error loading timezone"))
				Expect(result).To(BeNil())
			})

			It("should error on invalid grace period", func() {
				invalidPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:        []string{"all"},
							StartTime:   "08:00",
							EndTime:     "18:00",
							GracePeriod: ptr.To("invalid-duration"),
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(invalidPeriod)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error parsing grace period"))
				Expect(result).To(BeNil())
			})
		})

		Context("with edge cases", func() {
			It("should handle nil minReplicas (defaults to 1)", func() {
				periodWithNilMin := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: nil,
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(periodWithNilMin)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.MinReplicas).To(Equal(int32(1)))
			})

			It("should handle nil maxReplicas (defaults to minReplicas)", func() {
				periodWithNilMax := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(3)),
					MaxReplicas: nil,
				}

				result, err := period.New(periodWithNilMax)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.MaxReplicas).To(Equal(int32(3)))
			})

			It("should handle nil grace period (defaults to 0s)", func() {
				periodWithNilGrace := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:        []string{"all"},
							StartTime:   "08:00",
							EndTime:     "18:00",
							GracePeriod: nil,
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(periodWithNilGrace)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.GracePeriod).To(Equal(time.Duration(0)))
			})

			It("should handle end time 00:00 (converts to 23:59)", func() {
				periodWithZeroEnd := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "08:00",
							EndTime:   "00:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(periodWithZeroEnd)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Period.EndTime).To(Equal("23:59"))
			})

			It("should handle single day with short notation", func() {
				periodWithShortDay := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"mon"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(periodWithShortDay)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Period.Days).To(Equal([]string{"mon"}))
			})

			It("should handle day with 3 characters", func() {
				periodWithThreeCharDay := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"tue"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(periodWithThreeCharDay)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Period.Days).To(Equal([]string{"tue"}))
			})
		})

		Context("with different period types", func() {
			It("should handle restore type", func() {
				restorePeriod := &common.ScalerPeriod{
					Type: "restore",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(restorePeriod)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Type).To(Equal("restore"))
			})

			It("should handle up type", func() {
				upPeriod := &common.ScalerPeriod{
					Type: "up",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(upPeriod)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Type).To(Equal("up"))
			})

			It("should handle down type", func() {
				downPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(downPeriod)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Type).To(Equal("down"))
			})
		})

		Context("with complex day configurations", func() {
			It("should handle multiple days", func() {
				multiDayPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"mon", "wed", "fri", "sun"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(multiDayPeriod)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Period.Days).To(HaveLen(4))
				Expect(result.Period.Days).To(ContainElements("mon", "wed", "fri", "sun"))
			})

			It("should handle all days", func() {
				allDaysPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "08:00",
							EndTime:   "18:00",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(allDaysPeriod)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Period.Days).To(Equal([]string{"all"}))
			})
		})

		Context("with time parsing edge cases", func() {
			It("should handle time with minutes", func() {
				timeWithMinutes := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "08:30",
							EndTime:   "18:15",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(timeWithMinutes)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Period.StartTime).To(Equal("08:30"))
				Expect(result.Period.EndTime).To(Equal("18:15"))
			})

			It("should handle midnight times", func() {
				midnightPeriod := &common.ScalerPeriod{
					Type: "down",
					Time: common.TimePeriod{
						Recurring: &common.RecurringPeriod{
							Days:      []string{"all"},
							StartTime: "00:00",
							EndTime:   "23:59",
						},
					},
					MinReplicas: ptr.To(int32(1)),
					MaxReplicas: ptr.To(int32(5)),
				}

				result, err := period.New(midnightPeriod)

				Expect(err).ToNot(HaveOccurred())
				Expect(result.Period.StartTime).To(Equal("00:00"))
				Expect(result.Period.EndTime).To(Equal("23:59"))
			})
		})
	})
})
