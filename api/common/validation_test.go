package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestScalerPeriod_Validate(t *testing.T) {
	tests := []struct {
		name    string
		period  ScalerPeriod
		wantErr error
	}{
		{
			name: "valid recurring period",
			period: ScalerPeriod{
				Type: PeriodTypeDown,
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday, DayFriday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "valid fixed period",
			period: ScalerPeriod{
				Type: PeriodTypeUp,
				Time: TimePeriod{
					Fixed: &FixedPeriod{
						StartTime: "2024-01-01 08:00:00",
						EndTime:   "2024-01-01 18:00:00",
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "invalid period type",
			period: ScalerPeriod{
				Type: PeriodType("invalid"),
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
				},
			},
			wantErr: ErrInvalidPeriodType,
		},
		{
			name: "missing time",
			period: ScalerPeriod{
				Type: PeriodTypeDown,
				Time: TimePeriod{},
			},
			wantErr: ErrTimeMissing,
		},
		{
			name: "both recurring and fixed",
			period: ScalerPeriod{
				Type: PeriodTypeDown,
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
					Fixed: &FixedPeriod{
						StartTime: "2024-01-01 08:00:00",
						EndTime:   "2024-01-01 18:00:00",
					},
				},
			},
			wantErr: ErrTimeBothSet,
		},
		{
			name: "min greater than max",
			period: ScalerPeriod{
				Type: PeriodTypeDown,
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
				},
				MinReplicas: ptr.To(int32(5)),
				MaxReplicas: ptr.To(int32(2)),
			},
			wantErr: ErrMinGreaterThanMax,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.period.Validate()
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

func TestRecurringPeriod_Validate(t *testing.T) {
	tests := []struct {
		name    string
		period  RecurringPeriod
		wantErr error
	}{
		{
			name: "valid",
			period: RecurringPeriod{
				Days:      []DayOfWeek{DayMonday, DayWednesday},
				StartTime: "08:00",
				EndTime:   "18:00",
			},
			wantErr: nil,
		},
		{
			name: "empty days",
			period: RecurringPeriod{
				Days:      []DayOfWeek{},
				StartTime: "08:00",
				EndTime:   "18:00",
			},
			wantErr: ErrDaysEmpty,
		},
		{
			name: "valid DayAll wildcard",
			period: RecurringPeriod{
				Days:      []DayOfWeek{DayAll},
				StartTime: "08:00",
				EndTime:   "18:00",
			},
			wantErr: nil,
		},
		{
			name: "valid full day name",
			period: RecurringPeriod{
				Days:      []DayOfWeek{DayOfWeek("monday"), DayOfWeek("Friday")},
				StartTime: "08:00",
				EndTime:   "18:00",
			},
			wantErr: nil,
		},
		{
			name: "valid uppercase day",
			period: RecurringPeriod{
				Days:      []DayOfWeek{DayOfWeek("WEDNESDAY")},
				StartTime: "08:00",
				EndTime:   "18:00",
			},
			wantErr: nil,
		},
		{
			name: "too short day",
			period: RecurringPeriod{
				Days:      []DayOfWeek{DayOfWeek("mo")},
				StartTime: "08:00",
				EndTime:   "18:00",
			},
			wantErr: ErrInvalidDay,
		},
		{
			name: "invalid day",
			period: RecurringPeriod{
				Days:      []DayOfWeek{DayOfWeek("invalid")},
				StartTime: "08:00",
				EndTime:   "18:00",
			},
			wantErr: ErrInvalidDay,
		},
		{
			name: "missing start time",
			period: RecurringPeriod{
				Days:    []DayOfWeek{DayMonday},
				EndTime: "18:00",
			},
			wantErr: ErrStartTimeRequired,
		},
		{
			name: "missing end time",
			period: RecurringPeriod{
				Days:      []DayOfWeek{DayMonday},
				StartTime: "08:00",
			},
			wantErr: ErrEndTimeRequired,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.period.Validate()
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

func Test_isValidDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		day  DayOfWeek
		want bool
	}{
		// All standard constants should be valid
		{name: "DayMonday constant", day: DayMonday, want: true},
		{name: "DayTuesday constant", day: DayTuesday, want: true},
		{name: "DayWednesday constant", day: DayWednesday, want: true},
		{name: "DayThursday constant", day: DayThursday, want: true},
		{name: "DayFriday constant", day: DayFriday, want: true},
		{name: "DaySaturday constant", day: DaySaturday, want: true},
		{name: "DaySunday constant", day: DaySunday, want: true},

		// DayAll ("all") is a valid wildcard matching every day of the week
		{name: "DayAll constant", day: DayAll, want: true},

		// Full day names (case-insensitive)
		{name: "full lowercase monday", day: "monday", want: true},
		{name: "full lowercase saturday", day: "saturday", want: true},
		{name: "full uppercase FRIDAY", day: "FRIDAY", want: true},
		{name: "mixed case Wednesday", day: "Wednesday", want: true},
		{name: "mixed case ThUrSdAy", day: "ThUrSdAy", want: true},

		// Exactly 3-char prefixes
		{name: "3-char mon", day: "mon", want: true},
		{name: "3-char tue", day: "tue", want: true},
		{name: "3-char wed", day: "wed", want: true},
		{name: "3-char thu", day: "thu", want: true},
		{name: "3-char fri", day: "fri", want: true},
		{name: "3-char sat", day: "sat", want: true},
		{name: "3-char sun", day: "sun", want: true},
		{name: "3-char uppercase MON", day: "MON", want: true},
		{name: "3-char uppercase TUE", day: "TUE", want: true},

		// Too short strings
		{name: "empty string", day: "", want: false},
		{name: "single char", day: "m", want: false},
		{name: "two chars mo", day: "mo", want: false},
		{name: "two chars fr", day: "fr", want: false},

		// Invalid day prefixes (3+ chars but not a valid day)
		{name: "invalid 3-char abc", day: "abc", want: false},
		{name: "invalid day name", day: "notaday", want: false},
		{name: "numeric string", day: "123", want: false},
		{name: "spaces", day: "   ", want: false},
		{name: "special characters", day: "m@n", want: false},

		// Unicode edge cases
		{name: "unicode chars", day: DayOfWeek("\u006d\u006f\u006e"), want: true},            // "mon" in unicode escapes
		{name: "non-ascii chars", day: DayOfWeek("m\u00f6n"), want: false},                   // "mon" with umlaut o
		{name: "multibyte unicode", day: DayOfWeek("\xe6\x9c\x88\xe6\x9b\x9c"), want: false}, // Japanese for Monday
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isValidDay(tc.day)
			assert.Equal(t, tc.want, got, "isValidDay(%q)", tc.day)
		})
	}
}

func TestScalerPeriod_Validate_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		period  ScalerPeriod
		wantErr error
	}{
		{
			name: "empty period type",
			period: ScalerPeriod{
				Type: PeriodType(""),
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
				},
			},
			wantErr: ErrInvalidPeriodType,
		},
		{
			name: "period type with whitespace",
			period: ScalerPeriod{
				Type: PeriodType(" up "),
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
				},
			},
			wantErr: ErrInvalidPeriodType,
		},
		{
			name: "period type uppercase UP",
			period: ScalerPeriod{
				Type: PeriodType("UP"),
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
				},
			},
			wantErr: ErrInvalidPeriodType,
		},
		{
			name: "min equals max replicas",
			period: ScalerPeriod{
				Type: PeriodTypeDown,
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
				},
				MinReplicas: ptr.To(int32(3)),
				MaxReplicas: ptr.To(int32(3)),
			},
			wantErr: nil,
		},
		{
			name: "only minReplicas set",
			period: ScalerPeriod{
				Type: PeriodTypeUp,
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
				},
				MinReplicas: ptr.To(int32(5)),
			},
			wantErr: nil,
		},
		{
			name: "only maxReplicas set",
			period: ScalerPeriod{
				Type: PeriodTypeUp,
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
				},
				MaxReplicas: ptr.To(int32(5)),
			},
			wantErr: nil,
		},
		{
			name: "zero replicas",
			period: ScalerPeriod{
				Type: PeriodTypeDown,
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days:      []DayOfWeek{DayMonday},
						StartTime: "08:00",
						EndTime:   "18:00",
					},
				},
				MinReplicas: ptr.To(int32(0)),
				MaxReplicas: ptr.To(int32(0)),
			},
			wantErr: nil,
		},
		{
			name: "recurring validation error propagates through ScalerPeriod",
			period: ScalerPeriod{
				Type: PeriodTypeDown,
				Time: TimePeriod{
					Recurring: &RecurringPeriod{
						Days: []DayOfWeek{},
					},
				},
			},
			wantErr: ErrDaysEmpty,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.period.Validate()
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

func TestRecurringPeriod_Validate_AllDayConstants(t *testing.T) {
	t.Parallel()

	// Verify all 7 standard day constants pass validation individually
	days := []DayOfWeek{DayMonday, DayTuesday, DayWednesday, DayThursday, DayFriday, DaySaturday, DaySunday}
	for _, day := range days {
		t.Run(string(day), func(t *testing.T) {
			t.Parallel()

			period := RecurringPeriod{
				Days:      []DayOfWeek{day},
				StartTime: "09:00",
				EndTime:   "17:00",
			}
			require.NoError(t, period.Validate())
		})
	}
}

func TestRecurringPeriod_Validate_AllDaysCombined(t *testing.T) {
	t.Parallel()

	period := RecurringPeriod{
		Days:      []DayOfWeek{DayMonday, DayTuesday, DayWednesday, DayThursday, DayFriday, DaySaturday, DaySunday},
		StartTime: "09:00",
		EndTime:   "17:00",
	}
	require.NoError(t, period.Validate())
}

func TestTimePeriod_Validate(t *testing.T) {
	tests := []struct {
		name    string
		period  TimePeriod
		wantErr error
	}{
		{
			name: "valid recurring",
			period: TimePeriod{
				Recurring: &RecurringPeriod{
					Days:      []DayOfWeek{DayMonday},
					StartTime: "08:00",
					EndTime:   "18:00",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid fixed",
			period: TimePeriod{
				Fixed: &FixedPeriod{
					StartTime: "2024-01-01 08:00:00",
					EndTime:   "2024-01-01 18:00:00",
				},
			},
			wantErr: nil,
		},
		{
			name:    "neither set",
			period:  TimePeriod{},
			wantErr: ErrTimeMissing,
		},
		{
			name: "both set",
			period: TimePeriod{
				Recurring: &RecurringPeriod{
					Days:      []DayOfWeek{DayMonday},
					StartTime: "08:00",
					EndTime:   "18:00",
				},
				Fixed: &FixedPeriod{
					StartTime: "2024-01-01 08:00:00",
					EndTime:   "2024-01-01 18:00:00",
				},
			},
			wantErr: ErrTimeBothSet,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.period.Validate()
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}
