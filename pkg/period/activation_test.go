package period_test

import (
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

// fakeClock implements period.Clock with a fixed time for deterministic testing.
type fakeClock struct {
	now time.Time
}

func (f fakeClock) Now() time.Time { return f.now }

var _ = Describe("Period Activation (IsActive with time injection)", func() {

	// Helper to build a recurring ScalerPeriod.
	// All recurring periods default to UTC timezone for deterministic tests.
	makeRecurring := func(days []common.DayOfWeek, start, end string, opts ...func(*common.RecurringPeriod)) *common.ScalerPeriod {
		rp := &common.RecurringPeriod{
			Days:      days,
			StartTime: start,
			EndTime:   end,
			Timezone:  ptr.To("UTC"),
		}
		for _, o := range opts {
			o(rp)
		}
		return &common.ScalerPeriod{
			Type:        common.PeriodTypeDown,
			Time:        common.TimePeriod{Recurring: rp},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(5)),
		}
	}

	// Helper to build a fixed ScalerPeriod.
	// All fixed periods default to UTC timezone for deterministic tests.
	makeFixed := func(start, end string, opts ...func(*common.FixedPeriod)) *common.ScalerPeriod {
		fp := &common.FixedPeriod{
			StartTime: start,
			EndTime:   end,
			Timezone:  ptr.To("UTC"),
		}
		for _, o := range opts {
			o(fp)
		}
		return &common.ScalerPeriod{
			Type:        common.PeriodTypeUp,
			Time:        common.TimePeriod{Fixed: fp},
			MinReplicas: ptr.To(int32(1)),
			MaxReplicas: ptr.To(int32(5)),
		}
	}

	withReverse := func(rp *common.RecurringPeriod) { rp.Reverse = ptr.To(true) }
	withOnce := func(rp *common.RecurringPeriod) { rp.Once = ptr.To(true) }
	withTimezone := func(tz string) func(*common.RecurringPeriod) {
		return func(rp *common.RecurringPeriod) { rp.Timezone = ptr.To(tz) }
	}
	withFixedTimezone := func(tz string) func(*common.FixedPeriod) {
		return func(fp *common.FixedPeriod) { fp.Timezone = ptr.To(tz) }
	}
	withFixedOnce := func(fp *common.FixedPeriod) { fp.Once = ptr.To(true) }

	type testCase struct {
		name     string
		now      time.Time
		period   *common.ScalerPeriod
		expected bool
	}

	// UTC location for building test times.
	utc := time.UTC

	// Monday 2026-03-09 in UTC
	monday := func(h, m int) time.Time {
		return time.Date(2026, 3, 9, h, m, 0, 0, utc)
	}
	// Tuesday 2026-03-10 in UTC
	tuesday := func(h, m int) time.Time {
		return time.Date(2026, 3, 10, h, m, 0, 0, utc)
	}

	DescribeTable("recurring period activation",
		func(tc testCase) {
			clock := fakeClock{now: tc.now}
			result, err := period.NewWithClock(tc.period, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(Equal(tc.expected), "expected IsActive=%v for %s", tc.expected, tc.name)
		},

		// 1. Recurring period active during the window
		Entry("active during window (Mon 10:00, window 08:00-18:00 on Mon)", testCase{
			name:     "active-during-window",
			now:      monday(10, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00"),
			expected: true,
		}),

		// 2. Recurring period inactive outside the window (before start)
		Entry("inactive before window (Mon 07:00, window 08:00-18:00)", testCase{
			name:     "inactive-before-window",
			now:      monday(7, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00"),
			expected: false,
		}),

		// Recurring period inactive outside the window (after end)
		Entry("inactive after window (Mon 19:00, window 08:00-18:00)", testCase{
			name:     "inactive-after-window",
			now:      monday(19, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00"),
			expected: false,
		}),

		// 3. Recurring period inactive on wrong day
		Entry("inactive on wrong day (Tue, window only Mon)", testCase{
			name:     "inactive-wrong-day",
			now:      tuesday(10, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00"),
			expected: false,
		}),

		// 11. "all" day keyword — active any day
		Entry("all days active (Mon 10:00, days=all)", testCase{
			name:     "all-days-mon",
			now:      monday(10, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayAll}, "08:00", "18:00"),
			expected: true,
		}),
		Entry("all days active (Tue 10:00, days=all)", testCase{
			name:     "all-days-tue",
			now:      tuesday(10, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayAll}, "08:00", "18:00"),
			expected: true,
		}),

		// 9. Reverse period behavior
		Entry("reverse: inactive during normal window (Mon 10:00)", testCase{
			name:     "reverse-inactive-during-window",
			now:      monday(10, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00", withReverse),
			expected: false,
		}),
		Entry("reverse: active outside normal window (Mon 07:00)", testCase{
			name:     "reverse-active-before-window",
			now:      monday(7, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00", withReverse),
			expected: true,
		}),
		Entry("reverse: active after normal window (Mon 19:00)", testCase{
			name:     "reverse-active-after-window",
			now:      monday(19, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00", withReverse),
			expected: true,
		}),

		// 12. Edge cases: exactly at start time and exactly at end time
		// isActive uses After(start) && Before(end), so exactly at start => NOT active
		Entry("exactly at start time (Mon 08:00:00)", testCase{
			name:     "exactly-at-start",
			now:      monday(8, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00"),
			expected: false,
		}),
		// End time 18:00 becomes 18:00:59; now=18:00:00 is Before(18:00:59) and After(08:00) => active
		Entry("exactly at end time (Mon 18:00:00, endTime inclusive +59s)", testCase{
			name:     "exactly-at-end",
			now:      monday(18, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00"),
			expected: true,
		}),
		// One second after start => active
		Entry("one second after start (Mon 08:00:01)", testCase{
			name: "one-second-after-start",
			now:  time.Date(2026, 3, 9, 8, 0, 1, 0, utc),
			period: makeRecurring(
				[]common.DayOfWeek{common.DayMonday}, "08:00", "18:00",
			),
			expected: true,
		}),

		// Multiple days in the list
		Entry("multiple days list, matching (Wed 12:00)", testCase{
			name: "multi-day-match",
			now:  time.Date(2026, 3, 11, 12, 0, 0, 0, utc), // Wednesday
			period: makeRecurring(
				[]common.DayOfWeek{common.DayMonday, common.DayWednesday, common.DayFriday},
				"08:00", "18:00",
			),
			expected: true,
		}),
		Entry("multiple days list, not matching (Thu 12:00)", testCase{
			name: "multi-day-no-match",
			now:  time.Date(2026, 3, 12, 12, 0, 0, 0, utc), // Thursday
			period: makeRecurring(
				[]common.DayOfWeek{common.DayMonday, common.DayWednesday, common.DayFriday},
				"08:00", "18:00",
			),
			expected: false,
		}),

		// EndTime 00:00 treated as 23:59 (+59s = 23:59:59)
		Entry("endTime 00:00 means end of day, active at 22:00", testCase{
			name:     "end-midnight-active",
			now:      monday(22, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "00:00"),
			expected: true,
		}),

		// Sunday handling (index 7 mapped to 0)
		Entry("Sunday period active on Sunday", testCase{
			name: "sunday-active",
			now:  time.Date(2026, 3, 8, 12, 0, 0, 0, utc), // Sunday
			period: makeRecurring(
				[]common.DayOfWeek{common.DaySunday}, "08:00", "18:00",
			),
			expected: true,
		}),
		Entry("Sunday period inactive on Monday", testCase{
			name: "sunday-inactive-on-monday",
			now:  monday(12, 0),
			period: makeRecurring(
				[]common.DayOfWeek{common.DaySunday}, "08:00", "18:00",
			),
			expected: false,
		}),
	)

	DescribeTable("fixed period activation",
		func(tc testCase) {
			clock := fakeClock{now: tc.now}
			result, err := period.NewWithClock(tc.period, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(Equal(tc.expected), "expected IsActive=%v for %s", tc.expected, tc.name)
		},

		// 4. Fixed period active during window
		Entry("fixed active during window", testCase{
			name:     "fixed-active",
			now:      time.Date(2026, 3, 9, 14, 0, 0, 0, utc),
			period:   makeFixed("2026-03-09 08:00:00", "2026-03-09 18:00:00"),
			expected: true,
		}),

		// 5. Fixed period inactive before start
		Entry("fixed inactive before start", testCase{
			name:     "fixed-before-start",
			now:      time.Date(2026, 3, 9, 6, 0, 0, 0, utc),
			period:   makeFixed("2026-03-09 08:00:00", "2026-03-09 18:00:00"),
			expected: false,
		}),

		// 6. Fixed period inactive after end
		Entry("fixed inactive after end", testCase{
			name:     "fixed-after-end",
			now:      time.Date(2026, 3, 9, 19, 0, 0, 0, utc),
			period:   makeFixed("2026-03-09 08:00:00", "2026-03-09 18:00:00"),
			expected: false,
		}),

		// Fixed period spanning multiple days
		Entry("fixed multi-day active on second day", testCase{
			name:     "fixed-multi-day-active",
			now:      time.Date(2026, 3, 10, 10, 0, 0, 0, utc),
			period:   makeFixed("2026-03-09 08:00:00", "2026-03-10 18:00:00"),
			expected: true,
		}),

		// Fixed with timezone
		Entry("fixed with timezone active", testCase{
			name: "fixed-tz-active",
			// 14:00 UTC = 15:00 Europe/Paris (CET +1)
			now:      time.Date(2026, 3, 9, 14, 0, 0, 0, utc),
			period:   makeFixed("2026-03-09 08:00:00", "2026-03-09 18:00:00", withFixedTimezone("Europe/Paris")),
			expected: true,
		}),

		// Fixed period exactly at start (After is strict => false)
		Entry("fixed exactly at start", testCase{
			name:     "fixed-exactly-start",
			now:      time.Date(2026, 3, 9, 8, 0, 0, 0, utc),
			period:   makeFixed("2026-03-09 08:00:00", "2026-03-09 18:00:00"),
			expected: false,
		}),
	)

	DescribeTable("once flag propagation",
		func(tc testCase) {
			clock := fakeClock{now: tc.now}
			result, err := period.NewWithClock(tc.period, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(Equal(tc.expected))
		},

		// 8. Once period behavior — once flag is propagated; IsActive still determined by time
		Entry("recurring once active during window", testCase{
			name:     "once-active",
			now:      monday(10, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00", withOnce),
			expected: true,
		}),
		Entry("recurring once inactive outside window", testCase{
			name:     "once-inactive",
			now:      monday(7, 0),
			period:   makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00", withOnce),
			expected: false,
		}),
	)

	Describe("once flag value on result", func() {
		It("should propagate once=true to result", func() {
			clock := fakeClock{now: monday(10, 0)}
			sp := makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:00", "18:00", withOnce)
			result, err := period.NewWithClock(sp, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Once).ToNot(BeNil())
			Expect(*result.Once).To(BeTrue())
		})

		It("should propagate once=false to result", func() {
			clock := fakeClock{now: monday(10, 0)}
			rp := &common.RecurringPeriod{
				Days:      []common.DayOfWeek{common.DayMonday},
				StartTime: "08:00",
				EndTime:   "18:00",
				Once:      ptr.To(false),
				Timezone:  ptr.To("UTC"),
			}
			sp := &common.ScalerPeriod{
				Type:        common.PeriodTypeDown,
				Time:        common.TimePeriod{Recurring: rp},
				MinReplicas: ptr.To(int32(1)),
				MaxReplicas: ptr.To(int32(5)),
			}
			result, err := period.NewWithClock(sp, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Once).ToNot(BeNil())
			Expect(*result.Once).To(BeFalse())
		})

		It("should propagate fixed once=true to result", func() {
			clock := fakeClock{now: time.Date(2026, 3, 9, 10, 0, 0, 0, utc)}
			sp := makeFixed("2026-03-09 08:00:00", "2026-03-09 18:00:00", withFixedOnce)
			result, err := period.NewWithClock(sp, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Once).ToNot(BeNil())
			Expect(*result.Once).To(BeTrue())
		})
	})

	DescribeTable("timezone handling",
		func(tc testCase) {
			clock := fakeClock{now: tc.now}
			result, err := period.NewWithClock(tc.period, clock)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.IsActive).To(Equal(tc.expected), "expected IsActive=%v for %s", tc.expected, tc.name)
		},

		// 10. Timezone handling
		// Window 08:00-18:00 Europe/Paris. At 09:00 UTC = 10:00 Paris => active
		Entry("timezone Europe/Paris active (09:00 UTC = 10:00 Paris)", testCase{
			name: "tz-paris-active",
			now:  time.Date(2026, 1, 15, 9, 0, 0, 0, utc), // Thu, Jan
			period: makeRecurring(
				[]common.DayOfWeek{common.DayThursday}, "08:00", "18:00",
				withTimezone("Europe/Paris"),
			),
			expected: true,
		}),
		// Window 08:00-18:00 Europe/Paris. At 06:00 UTC = 07:00 Paris => inactive
		Entry("timezone Europe/Paris inactive (06:00 UTC = 07:00 Paris)", testCase{
			name: "tz-paris-inactive",
			now:  time.Date(2026, 1, 15, 6, 0, 0, 0, utc), // Thu
			period: makeRecurring(
				[]common.DayOfWeek{common.DayThursday}, "08:00", "18:00",
				withTimezone("Europe/Paris"),
			),
			expected: false,
		}),
		// Timezone can shift the day: 23:30 UTC on Monday = 00:30 Tue in Paris
		// Period on Tuesday 00:00-06:00 Paris => now is 00:30 Paris Tue => active
		// Note: endTime 00:00 is converted to 23:59, so we use 00:01 start.
		Entry("timezone shifts day boundary (23:30 UTC Mon = 00:30 Tue Paris)", testCase{
			name: "tz-day-shift",
			now:  time.Date(2026, 1, 12, 23, 30, 0, 0, utc), // Mon UTC => Tue Paris
			period: makeRecurring(
				[]common.DayOfWeek{common.DayTuesday}, "00:01", "06:00",
				withTimezone("Europe/Paris"),
			),
			expected: true,
		}),

		// US timezone: 08:00-18:00 America/New_York
		// 20:00 UTC on Mon = 15:00 ET => active
		Entry("timezone America/New_York active (20:00 UTC = 15:00 ET)", testCase{
			name: "tz-ny-active",
			now:  time.Date(2026, 1, 12, 20, 0, 0, 0, utc), // Mon
			period: makeRecurring(
				[]common.DayOfWeek{common.DayMonday}, "08:00", "18:00",
				withTimezone("America/New_York"),
			),
			expected: true,
		}),
	)

	Describe("overnight recurring period (start > end) returns error", func() {
		// The code explicitly does NOT support overnight periods for recurring.
		// startTime > endTime => ErrStartAfterEnd
		It("should return ErrStartAfterEnd for 22:00-06:00", func() {
			clock := fakeClock{now: monday(23, 0)}
			sp := makeRecurring([]common.DayOfWeek{common.DayMonday}, "22:00", "06:00")
			_, err := period.NewWithClock(sp, clock)
			Expect(err).To(Equal(period.ErrStartAfterEnd))
		})
	})

	Describe("reverse as overnight workaround", func() {
		// To achieve 22:00-07:00 effect, use reverse on 07:00-22:00
		// reverse=true: isActive = !( After(07:00) && Before(22:00:59) )
		// At 23:00 Mon: After(07:00)=true, Before(22:00:59)=false => normal=false => reversed=true
		// At 02:00 Mon: After(07:00)=false => normal=false => reversed=true
		// At 10:00 Mon: After(07:00)=true, Before(22:00:59)=true => normal=true => reversed=false

		DescribeTable("reverse simulates overnight",
			func(tc testCase) {
				clock := fakeClock{now: tc.now}
				result, err := period.NewWithClock(tc.period, clock)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.IsActive).To(Equal(tc.expected), "expected IsActive=%v for %s", tc.expected, tc.name)
			},

			// 7. Overnight via reverse: 07:00-22:00 reverse=true => active outside 07:00-22:00
			Entry("reverse overnight: active at 23:00", testCase{
				name: "reverse-overnight-23",
				now:  monday(23, 0),
				period: makeRecurring(
					[]common.DayOfWeek{common.DayMonday}, "07:00", "22:00", withReverse,
				),
				expected: true,
			}),
			Entry("reverse overnight: active at 02:00", testCase{
				name: "reverse-overnight-02",
				now:  monday(2, 0),
				period: makeRecurring(
					[]common.DayOfWeek{common.DayMonday}, "07:00", "22:00", withReverse,
				),
				expected: true,
			}),
			Entry("reverse overnight: inactive at 10:00 (inside daytime window)", testCase{
				name: "reverse-overnight-10",
				now:  monday(10, 0),
				period: makeRecurring(
					[]common.DayOfWeek{common.DayMonday}, "07:00", "22:00", withReverse,
				),
				expected: false,
			}),
		)
	})

	Describe("start and end time boundaries", func() {
		It("should set correct StartTime and EndTime on the result", func() {
			clock := fakeClock{now: monday(12, 0)}
			sp := makeRecurring([]common.DayOfWeek{common.DayMonday}, "08:30", "17:45")
			result, err := period.NewWithClock(sp, clock)
			Expect(err).ToNot(HaveOccurred())

			// StartTime should be 2026-03-09 08:30:00 UTC
			Expect(result.StartTime.Hour()).To(Equal(8))
			Expect(result.StartTime.Minute()).To(Equal(30))
			// EndTime should be 2026-03-09 17:45:59 UTC (inclusive +59s)
			Expect(result.EndTime.Hour()).To(Equal(17))
			Expect(result.EndTime.Minute()).To(Equal(45))
			Expect(result.EndTime.Second()).To(Equal(59))
		})
	})
})
