package period

import (
	"crypto/sha1" //nolint: gosec
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	kubecloudscalerv1alpha1 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha1"
	"k8s.io/utils/ptr"
)

func New(period *kubecloudscalerv1alpha1.ScalerPeriod) (*Period, error) {
	var err error

	curPeriod := &Period{
		IsActive: false,
		Type:     period.Type,
	}

	// first check for the fixed period by converting to recuuring one
	convertedPeriod := convertFixedToRecurring(period.Time.Fixed)
	periodType := PeriodFixedName

	if convertedPeriod == nil {
		convertedPeriod = period.Time.Recurring
		periodType = PeriodRecurringName
	}

	curPeriod.MinReplicas = ptr.Deref(period.MinReplicas, int32(1))
	curPeriod.MaxReplicas = ptr.Deref(period.MaxReplicas, curPeriod.MinReplicas)

	if curPeriod.MinReplicas > curPeriod.MaxReplicas {
		return nil, ErrMinReplicasGreaterThanMax
	}

	curPeriod.IsActive, curPeriod.GetStartTime, curPeriod.GetEndTime, curPeriod.Once, err = isPeriodActive(
		periodType,
		convertedPeriod,
	)
	if err != nil {
		return nil, err
	}

	curPeriod.GracePeriod, err = time.ParseDuration(ptr.Deref(convertedPeriod.GracePeriod, defaultGracePeriod))
	if err != nil {
		return nil, fmt.Errorf("error parsing grace period: %w", err)
	}

	curPeriod.Period = convertedPeriod

	periodData, err := json.Marshal(period)
	if err != nil {
		return nil, fmt.Errorf("error marshalling period: %w", err)
	}

	curPeriod.Hash = fmt.Sprintf("%x", sha1.Sum(periodData)) //nolint: gosec

	return curPeriod, nil
}

func isDay(day string, localTime *time.Time) (bool, error) {
	if day == allDays {
		return true, nil
	}

	localDay := int(localTime.Weekday())
	sanitizedDay := strings.ToLower(day)

	if strings.Count(day, "") <= dayStringLength {
		sanitizedDay = "bad"
	}

	// check if the day is valid
	// shorten the day value to 3 chars, lowered
	indexedDay := slices.Index(weekDays, sanitizedDay[:3])

	switch indexedDay {
	case -1:
		return false, fmt.Errorf("%w: %s", ErrBadDay, day)
	case 7:
		indexedDay = localDay
	}

	// are we in the right day?
	return indexedDay == localDay, nil
}

func getTime(period, periodType string, timeLocation *time.Location) (time.Time, error) {
	var (
		outTime time.Time
		err     error
	)

	switch periodType {
	case PeriodRecurringName:
		timeOnly, err := time.ParseInLocation(time.TimeOnly, period+":00", timeLocation)
		if err != nil {
			return timeOnly, ErrRecurringTimeFormat
		}

		localTime := time.Now().In(timeLocation)

		outTime = time.Date(
			localTime.Year(),
			localTime.Month(),
			localTime.Day(),
			timeOnly.Hour(),
			timeOnly.Minute(),
			0,
			0,
			timeLocation,
		)
	case PeriodFixedName:
		outTime, err = time.ParseInLocation(time.DateTime, period, timeLocation)
		if err != nil {
			return outTime, ErrFixedTimeFormat
		}
	default:
		return time.Time{}, fmt.Errorf("%w: %s", ErrUnknownPeriodType, periodType)
	}

	return outTime, nil
}

func isPeriodActive(
	periodType string,
	period *kubecloudscalerv1alpha1.RecurringPeriod,
) (bool, time.Time, time.Time, *bool, error) {
	var (
		err error
	)

	onDay := false
	timeLocation := time.Local

	if period.Timezone != nil {
		timeLocation, err = time.LoadLocation(ptr.Deref(period.Timezone, defaultTimezone))
		if err != nil {
			return false, time.Time{}, time.Time{}, nil, fmt.Errorf("error loading timezone: %w", err)
		}
	}

	localTime := time.Now().In(timeLocation)

	for _, day := range period.Days {
		var onDayErr error
		// check if we are in the right day
		onDay, onDayErr = isDay(day, &localTime)
		if onDayErr != nil {
			return false, time.Time{}, time.Time{}, nil, onDayErr
		}

		if onDay {
			break
		}
	}

	if !onDay {
		return onDay, time.Time{}, time.Time{}, nil, nil
	}

	startTime, err := getTime(period.StartTime, periodType, timeLocation)
	if err != nil {
		return false, time.Time{}, time.Time{}, nil, err
	}

	endTime, err := getTime(period.EndTime, periodType, timeLocation)
	if err != nil {
		return false, time.Time{}, time.Time{}, nil, err
	}

	if startTime.After(endTime) {
		return false, time.Time{}, time.Time{}, nil, ErrStartAfterEnd
	}

	isActive := localTime.After(startTime) && localTime.Before(endTime)
	if ptr.Deref(period.Reverse, false) {
		isActive = !isActive
	}

	return isActive, startTime, endTime, period.Once, nil
}

func convertFixedToRecurring(fixed *kubecloudscalerv1alpha1.FixedPeriod) *kubecloudscalerv1alpha1.RecurringPeriod {
	if fixed == nil {
		return nil
	}

	return &kubecloudscalerv1alpha1.RecurringPeriod{
		Days: []string{
			"all",
		},
		StartTime:   fixed.StartTime,
		EndTime:     fixed.EndTime,
		Timezone:    fixed.Timezone,
		Once:        fixed.Once,
		GracePeriod: fixed.GracePeriod,
	}
}
