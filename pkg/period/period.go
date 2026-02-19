// Package period provides period management functionality for the kubecloudscaler project.
package period

import (
	"crypto/sha1" //nolint:gosec // SHA1 is used for hash generation, not cryptographic security
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"k8s.io/utils/ptr"
)

const (
	// SundayIndex represents Sunday in the week days slice (index 7).
	SundayIndex = 7
	// EndTimeInclusiveSeconds is added to make end time inclusive.
	EndTimeInclusiveSeconds = 59
)

// New creates a new Period from the given ScalerPeriod configuration.
func New(period *common.ScalerPeriod) (*Period, error) {
	var err error

	curPeriod := &Period{
		IsActive: false,
		Type:     period.Type,
		Name:     period.Name,
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

	curPeriod.Hash = fmt.Sprintf("%x", sha1.Sum(periodData)) //nolint:gosec // SHA1 is used for hash generation, not cryptographic security

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
	case SundayIndex:
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

//nolint:gocyclo,gocritic // Period validation complexity acceptable, multiple returns needed
func isPeriodActive(
	periodType string,
	period *common.RecurringPeriod,
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

	endTimeStr := period.EndTime
	if endTimeStr == "00:00" {
		// if the end time is 00:00, it means the period ends at the end of the day
		endTimeStr = "23:59"
	}

	endTime, err := getTime(endTimeStr, periodType, timeLocation)
	if err != nil {
		return false, time.Time{}, time.Time{}, nil, err
	}

	endTime = endTime.Add(time.Second * EndTimeInclusiveSeconds) // end time is inclusive, so we add 59 seconds

	if startTime.After(endTime) {
		return false, time.Time{}, time.Time{}, nil, ErrStartAfterEnd
	}

	isActive := localTime.After(startTime) && localTime.Before(endTime)
	if ptr.Deref(period.Reverse, false) {
		isActive = !isActive
	}

	return isActive, startTime, endTime, period.Once, nil
}

func convertFixedToRecurring(fixed *common.FixedPeriod) *common.RecurringPeriod {
	if fixed == nil {
		return nil
	}

	return &common.RecurringPeriod{
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
