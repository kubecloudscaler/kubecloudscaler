package period_test

import (
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

// mockClock implements period.Clock for deterministic time control.
type mockClock struct {
	now time.Time
}

func (m mockClock) Now() time.Time { return m.now }

var _ = Describe("isDay via NewWithClock", func() {
	// We use NewWithClock to indirectly test isDay behavior:
	// if the day doesn't match, IsActive will be false (not an error).
	// if the day is invalid (too short or bad prefix), an error is returned.
	var basePeriod func(days []common.DayOfWeek) *common.ScalerPeriod

	BeforeEach(func() {
		basePeriod = func(days []common.DayOfWeek) *common.ScalerPeriod {
			return &common.ScalerPeriod{
				Type: common.PeriodTypeDown,
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						Days:      days,
						StartTime: "00:00",
						EndTime:   "23:59",
					},
				},
				MinReplicas: ptr.To(int32(1)),
				MaxReplicas: ptr.To(int32(5)),
			}
		}
	})

	DescribeTable("each standard DayOfWeek constant matches its weekday",
		func(day common.DayOfWeek, weekday time.Weekday) {
			// Pick a date that falls on the given weekday
			// 2024-01-01 is Monday; offset accordingly
			baseDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC) // Monday
			daysOffset := int(weekday) - int(time.Monday)
			if daysOffset < 0 {
				daysOffset += 7
			}
			targetDate := baseDate.AddDate(0, 0, daysOffset)
			clock := mockClock{now: targetDate}

			result, err := period.NewWithClock(basePeriod([]common.DayOfWeek{day}), clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(BeTrue(), "expected %s to be active on %s", day, targetDate.Weekday())
		},
		Entry("mon on Monday", common.DayMonday, time.Monday),
		Entry("tue on Tuesday", common.DayTuesday, time.Tuesday),
		Entry("wed on Wednesday", common.DayWednesday, time.Wednesday),
		Entry("thu on Thursday", common.DayThursday, time.Thursday),
		Entry("fri on Friday", common.DayFriday, time.Friday),
		Entry("sat on Saturday", common.DaySaturday, time.Saturday),
		Entry("sun on Sunday", common.DaySunday, time.Sunday),
	)

	DescribeTable("DayAll matches every weekday",
		func(weekday time.Weekday) {
			baseDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC) // Monday
			daysOffset := int(weekday) - int(time.Monday)
			if daysOffset < 0 {
				daysOffset += 7
			}
			targetDate := baseDate.AddDate(0, 0, daysOffset)
			clock := mockClock{now: targetDate}

			result, err := period.NewWithClock(basePeriod([]common.DayOfWeek{common.DayAll}), clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(BeTrue(), "expected 'all' to be active on %s", weekday)
		},
		Entry("Monday", time.Monday),
		Entry("Tuesday", time.Tuesday),
		Entry("Wednesday", time.Wednesday),
		Entry("Thursday", time.Thursday),
		Entry("Friday", time.Friday),
		Entry("Saturday", time.Saturday),
		Entry("Sunday", time.Sunday),
	)

	DescribeTable("day does not match different weekday",
		func(day common.DayOfWeek, wrongWeekday time.Weekday) {
			baseDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC) // Monday
			daysOffset := int(wrongWeekday) - int(time.Monday)
			if daysOffset < 0 {
				daysOffset += 7
			}
			targetDate := baseDate.AddDate(0, 0, daysOffset)
			clock := mockClock{now: targetDate}

			result, err := period.NewWithClock(basePeriod([]common.DayOfWeek{day}), clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(BeFalse(), "expected %s to NOT be active on %s", day, wrongWeekday)
		},
		Entry("mon on Tuesday", common.DayMonday, time.Tuesday),
		Entry("fri on Wednesday", common.DayFriday, time.Wednesday),
		Entry("sun on Monday", common.DaySunday, time.Monday),
		Entry("sat on Thursday", common.DaySaturday, time.Thursday),
	)

	DescribeTable("full day names are accepted (case-insensitive)",
		func(dayStr string, weekday time.Weekday) {
			baseDate := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
			daysOffset := int(weekday) - int(time.Monday)
			if daysOffset < 0 {
				daysOffset += 7
			}
			targetDate := baseDate.AddDate(0, 0, daysOffset)
			clock := mockClock{now: targetDate}

			result, err := period.NewWithClock(basePeriod([]common.DayOfWeek{common.DayOfWeek(dayStr)}), clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(BeTrue(), "expected %q to be active on %s", dayStr, weekday)
		},
		Entry("monday", "monday", time.Monday),
		Entry("Monday", "Monday", time.Monday),
		Entry("MONDAY", "MONDAY", time.Monday),
		Entry("tuesday", "tuesday", time.Tuesday),
		Entry("SUNDAY", "SUNDAY", time.Sunday),
		Entry("Saturday", "Saturday", time.Saturday),
	)

	DescribeTable("invalid day strings return errors",
		func(dayStr string) {
			clock := mockClock{now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}

			_, err := period.NewWithClock(basePeriod([]common.DayOfWeek{common.DayOfWeek(dayStr)}), clock)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid day notation"))
		},
		Entry("empty string", ""),
		Entry("single char", "m"),
		Entry("two chars", "mo"),
		Entry("invalid prefix", "xyz"),
		Entry("numeric", "123"),
	)

	Context("convertFixedToRecurring preserves all fields", func() {
		It("should preserve timezone, once, and gracePeriod from fixed period", func() {
			fixedPeriod := &common.ScalerPeriod{
				Type: common.PeriodTypeUp,
				Time: common.TimePeriod{
					Fixed: &common.FixedPeriod{
						StartTime:   "2024-01-01 08:00:00",
						EndTime:     "2024-01-01 18:00:00",
						Timezone:    ptr.To("Europe/Paris"),
						Once:        ptr.To(true),
						GracePeriod: ptr.To("30s"),
					},
				},
				MinReplicas: ptr.To(int32(2)),
				MaxReplicas: ptr.To(int32(8)),
			}

			clock := mockClock{now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
			result, err := period.NewWithClock(fixedPeriod, clock)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Spec.Timezone).To(Equal(ptr.To("Europe/Paris")))
			Expect(*result.Spec.Once).To(BeTrue())
			Expect(result.Spec.GracePeriod).To(Equal(ptr.To("30s")))
			Expect(result.Spec.Days).To(Equal([]common.DayOfWeek{common.DayAll}))
		})
	})

	Context("getTime via NewWithClock", func() {
		It("should error on unknown period type passed indirectly", func() {
			// This is tested indirectly; the period package doesn't expose getTime,
			// but we can test invalid time formats which exercise it
			invalidTimePeriod := &common.ScalerPeriod{
				Type: common.PeriodTypeDown,
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						Days:      []common.DayOfWeek{common.DayAll},
						StartTime: "25:99",
						EndTime:   "18:00",
					},
				},
				MinReplicas: ptr.To(int32(1)),
			}

			clock := mockClock{now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
			_, err := period.NewWithClock(invalidTimePeriod, clock)
			Expect(err).To(Equal(period.ErrRecurringTimeFormat))
		})

		It("should error on invalid end time format", func() {
			invalidEndTime := &common.ScalerPeriod{
				Type: common.PeriodTypeDown,
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						Days:      []common.DayOfWeek{common.DayAll},
						StartTime: "08:00",
						EndTime:   "not-a-time",
					},
				},
				MinReplicas: ptr.To(int32(1)),
			}

			clock := mockClock{now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
			_, err := period.NewWithClock(invalidEndTime, clock)
			Expect(err).To(Equal(period.ErrRecurringTimeFormat))
		})
	})

	Context("period active/inactive boundary conditions", func() {
		It("should be inactive before start time", func() {
			p := basePeriod([]common.DayOfWeek{common.DayAll})
			p.Time.Recurring.StartTime = "10:00"
			p.Time.Recurring.EndTime = "18:00"
			p.Time.Recurring.Timezone = ptr.To("UTC")

			clock := mockClock{now: time.Date(2024, 1, 1, 9, 59, 59, 0, time.UTC)}
			result, err := period.NewWithClock(p, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(BeFalse())
		})

		It("should be active after start time", func() {
			p := basePeriod([]common.DayOfWeek{common.DayAll})
			p.Time.Recurring.StartTime = "10:00"
			p.Time.Recurring.EndTime = "18:00"
			p.Time.Recurring.Timezone = ptr.To("UTC")

			clock := mockClock{now: time.Date(2024, 1, 1, 10, 0, 1, 0, time.UTC)}
			result, err := period.NewWithClock(p, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(BeTrue())
		})

		It("should be active just before end time (inclusive with 59s)", func() {
			p := basePeriod([]common.DayOfWeek{common.DayAll})
			p.Time.Recurring.StartTime = "10:00"
			p.Time.Recurring.EndTime = "18:00"
			p.Time.Recurring.Timezone = ptr.To("UTC")

			clock := mockClock{now: time.Date(2024, 1, 1, 18, 0, 58, 0, time.UTC)}
			result, err := period.NewWithClock(p, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(BeTrue())
		})

		It("should be inactive after end time plus 59 seconds", func() {
			p := basePeriod([]common.DayOfWeek{common.DayAll})
			p.Time.Recurring.StartTime = "10:00"
			p.Time.Recurring.EndTime = "18:00"
			p.Time.Recurring.Timezone = ptr.To("UTC")

			// End time is 18:00 + 59s = 18:00:59; at 18:01:00 should be inactive
			clock := mockClock{now: time.Date(2024, 1, 1, 18, 1, 0, 0, time.UTC)}
			result, err := period.NewWithClock(p, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(BeFalse())
		})
	})

	Context("reverse period", func() {
		It("should invert active status with reverse=true", func() {
			p := basePeriod([]common.DayOfWeek{common.DayAll})
			p.Time.Recurring.StartTime = "10:00"
			p.Time.Recurring.EndTime = "18:00"
			p.Time.Recurring.Timezone = ptr.To("UTC")
			p.Time.Recurring.Reverse = ptr.To(true)

			// During the period window — with reverse, this should be inactive
			clock := mockClock{now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
			result, err := period.NewWithClock(p, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(BeFalse())
		})

		It("should be active outside the window with reverse=true", func() {
			p := basePeriod([]common.DayOfWeek{common.DayAll})
			p.Time.Recurring.StartTime = "10:00"
			p.Time.Recurring.EndTime = "18:00"
			p.Time.Recurring.Timezone = ptr.To("UTC")
			p.Time.Recurring.Reverse = ptr.To(true)

			// Outside the period window — with reverse, this should be active
			clock := mockClock{now: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)}
			result, err := period.NewWithClock(p, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(BeTrue())
		})
	})

	Context("period name", func() {
		It("should preserve the period name", func() {
			p := basePeriod([]common.DayOfWeek{common.DayAll})
			p.Name = "business-hours"

			clock := mockClock{now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
			result, err := period.NewWithClock(p, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Name).To(Equal("business-hours"))
		})

		It("should handle empty period name", func() {
			p := basePeriod([]common.DayOfWeek{common.DayAll})
			p.Name = ""

			clock := mockClock{now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
			result, err := period.NewWithClock(p, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Name).To(BeEmpty())
		})
	})

	Context("hash generation", func() {
		It("should produce different hashes for different periods", func() {
			p1 := basePeriod([]common.DayOfWeek{common.DayMonday})
			p2 := basePeriod([]common.DayOfWeek{common.DayFriday})

			clock := mockClock{now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)} // Monday

			r1, err := period.NewWithClock(p1, clock)
			Expect(err).ToNot(HaveOccurred())

			// Use a Friday clock for p2 so it resolves
			fridayClock := mockClock{now: time.Date(2024, 1, 5, 12, 0, 0, 0, time.UTC)} // Friday
			r2, err := period.NewWithClock(p2, fridayClock)
			Expect(err).ToNot(HaveOccurred())

			Expect(r1.Hash).ToNot(Equal(r2.Hash))
		})

		It("should produce identical hashes for identical periods", func() {
			p1 := basePeriod([]common.DayOfWeek{common.DayAll})
			p2 := basePeriod([]common.DayOfWeek{common.DayAll})

			clock := mockClock{now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}

			r1, err := period.NewWithClock(p1, clock)
			Expect(err).ToNot(HaveOccurred())
			r2, err := period.NewWithClock(p2, clock)
			Expect(err).ToNot(HaveOccurred())

			Expect(r1.Hash).To(Equal(r2.Hash))
		})
	})
})
