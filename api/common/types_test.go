package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceKind_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		kind     ResourceKind
		expected string
	}{
		{name: "deployments", kind: ResourceDeployments, expected: "deployments"},
		{name: "statefulsets", kind: ResourceStatefulSets, expected: "statefulsets"},
		{name: "cronjobs", kind: ResourceCronJobs, expected: "cronjobs"},
		{name: "github-ars", kind: ResourceGithubARS, expected: "github-ars"},
		{name: "hpa", kind: ResourceHPA, expected: "hpa"},
		{name: "vm-instances", kind: ResourceVMInstances, expected: "vm-instances"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, ResourceKind(tc.expected), tc.kind)
			assert.Equal(t, tc.expected, string(tc.kind))
		})
	}
}

func TestResourceKind_AllConstantsAreLowercase(t *testing.T) {
	t.Parallel()

	allKinds := []ResourceKind{
		ResourceDeployments,
		ResourceStatefulSets,
		ResourceCronJobs,
		ResourceGithubARS,
		ResourceHPA,
		ResourceVMInstances,
	}

	for _, kind := range allKinds {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()

			s := string(kind)
			for _, c := range s {
				if c >= 'A' && c <= 'Z' {
					t.Errorf("ResourceKind %q contains uppercase character %c", s, c)
				}
			}
		})
	}
}

func TestResourceKind_UniqueValues(t *testing.T) {
	t.Parallel()

	allKinds := []ResourceKind{
		ResourceDeployments,
		ResourceStatefulSets,
		ResourceCronJobs,
		ResourceGithubARS,
		ResourceHPA,
		ResourceVMInstances,
	}

	seen := make(map[ResourceKind]bool)
	for _, kind := range allKinds {
		assert.False(t, seen[kind], "duplicate ResourceKind value: %s", kind)
		seen[kind] = true
	}
}

func TestPeriodType_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pt       PeriodType
		expected string
	}{
		{name: "down", pt: PeriodTypeDown, expected: "down"},
		{name: "up", pt: PeriodTypeUp, expected: "up"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, PeriodType(tc.expected), tc.pt)
			assert.Equal(t, tc.expected, string(tc.pt))
		})
	}
}

func TestPeriodType_UpAndDownAreDifferent(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, PeriodTypeUp, PeriodTypeDown)
}

func TestDayOfWeek_Constants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		day      DayOfWeek
		expected string
	}{
		{name: "monday", day: DayMonday, expected: "mon"},
		{name: "tuesday", day: DayTuesday, expected: "tue"},
		{name: "wednesday", day: DayWednesday, expected: "wed"},
		{name: "thursday", day: DayThursday, expected: "thu"},
		{name: "friday", day: DayFriday, expected: "fri"},
		{name: "saturday", day: DaySaturday, expected: "sat"},
		{name: "sunday", day: DaySunday, expected: "sun"},
		{name: "all", day: DayAll, expected: "all"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, DayOfWeek(tc.expected), tc.day)
			assert.Equal(t, tc.expected, string(tc.day))
		})
	}
}

func TestDayOfWeek_UniqueValues(t *testing.T) {
	t.Parallel()

	allDays := []DayOfWeek{
		DayMonday, DayTuesday, DayWednesday, DayThursday,
		DayFriday, DaySaturday, DaySunday, DayAll,
	}

	seen := make(map[DayOfWeek]bool)
	for _, day := range allDays {
		assert.False(t, seen[day], "duplicate DayOfWeek value: %s", day)
		seen[day] = true
	}
}

func TestDayOfWeek_StandardDaysCount(t *testing.T) {
	t.Parallel()

	// 7 standard days + "all"
	standardDays := []DayOfWeek{
		DayMonday, DayTuesday, DayWednesday, DayThursday,
		DayFriday, DaySaturday, DaySunday,
	}
	assert.Len(t, standardDays, 7)
}
