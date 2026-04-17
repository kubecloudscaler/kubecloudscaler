/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package service

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
)

var _ = Describe("TimeCalculatorService", func() {
	var (
		logger zerolog.Logger
		svc    *TimeCalculatorService
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		svc = NewTimeCalculatorService(&logger)
	})

	Describe("GetPeriodDuration", func() {
		It("should return the correct duration for a recurring period", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						StartTime: "09:00",
						EndTime:   "17:00",
					},
				},
			}

			duration, err := svc.GetPeriodDuration(period)

			Expect(err).ToNot(HaveOccurred())
			Expect(duration).To(Equal(8 * time.Hour))
		})

		It("should treat a recurring period that crosses midnight as spanning the next day", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						StartTime: "22:00",
						EndTime:   "02:00",
					},
				},
			}

			duration, err := svc.GetPeriodDuration(period)

			Expect(err).ToNot(HaveOccurred())
			Expect(duration).To(Equal(4 * time.Hour))
		})

		It("should treat an exactly-midnight cross-midnight recurring period correctly", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						StartTime: "23:30",
						EndTime:   "00:30",
					},
				},
			}

			duration, err := svc.GetPeriodDuration(period)

			Expect(err).ToNot(HaveOccurred())
			Expect(duration).To(Equal(1 * time.Hour))
		})

		It("should still reject end-before-start for fixed periods (full datetimes)", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Fixed: &common.FixedPeriod{
						StartTime: "2026-04-17 10:00:00",
						EndTime:   "2026-04-17 09:00:00",
					},
				},
			}

			_, err := svc.GetPeriodDuration(period)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("end time is before start time"))
		})
	})

	Describe("CalculatePeriodStartTime", func() {
		It("should return the start time offset by the given delay", func() {
			period := &common.ScalerPeriod{
				Time: common.TimePeriod{
					Recurring: &common.RecurringPeriod{
						StartTime: "09:00",
					},
				},
			}

			startTime, err := svc.CalculatePeriodStartTime(period, 1*time.Hour)

			Expect(err).ToNot(HaveOccurred())
			Expect(startTime.Hour()).To(Equal(10))
			Expect(startTime.Minute()).To(Equal(0))
		})
	})
})
