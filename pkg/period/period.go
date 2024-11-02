package period

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cloudscalerio/cloudscaler/api/common"
)

func New(period *common.ScalerPeriod) (*Period, error) {
	var err error

	timeLocation, err := time.LoadLocation(period.Time.Timezone)
	if err != nil {
		return &Period{}, err
	}

	localTime := time.Now().In(timeLocation)

	curPeriod := &Period{
		Period: period,
	}
	// check if the period is active
	curPeriod.IsActive, curPeriod.GetStartTime, curPeriod.GetEndTime, err = isPeriodActive(
		period,
		&localTime,
		timeLocation,
	)
	if err != nil {
		return nil, err
	}

	periodData, err := json.Marshal(period)
	if err != nil {
		return nil, err
	}

	curPeriod.Hash = fmt.Sprintf("%x", sha1.Sum(periodData))

	return curPeriod, nil
}

func isDay(day string, localTime *time.Time) (bool, error) {
	if day == allDays {
		return true, nil
	}

	localDay := int(localTime.Weekday())
	sanitizedDay := strings.ToLower(day)

	if strings.Count(day, "") <= 3 {
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

func checkTime(period string) ([]int, error) {
	periodSplit := strings.Split(period, ":")
	output := []int{}
	for _, val := range periodSplit {
		converted, err := strconv.Atoi(val)
		if err != nil {
			return output, ErrBadTime
		}

		output = append(output, converted)
	}

	return output, nil
}

func getTime(period string, localTime *time.Time, timeLocation *time.Location) (time.Time, error) {
	periodSplit, err := checkTime(period)
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(
		localTime.Year(),
		localTime.Month(),
		localTime.Day(),
		periodSplit[0],
		periodSplit[1],
		0,
		0,
		timeLocation,
	), nil
}

func isPeriodActive(
	period *common.ScalerPeriod,
	localTime *time.Time,
	timeLocation *time.Location,
) (bool, time.Time, time.Time, error) {
	onDay := false

	for _, day := range period.Time.Days {
		var onDayErr error
		// check if we are in the right day
		onDay, onDayErr = isDay(day, localTime)
		if onDayErr != nil {
			return false, time.Time{}, time.Time{}, onDayErr
		}

		if onDay {
			break
		}
	}

	if !onDay {
		return onDay, time.Time{}, time.Time{}, nil
	}

	startTime, err := getTime(period.Time.StartTime, localTime, timeLocation)
	if err != nil {
		return false, time.Time{}, time.Time{}, err
	}

	endTime, err := getTime(period.Time.EndTime, localTime, timeLocation)
	if err != nil {
		return false, time.Time{}, time.Time{}, err
	}

	return localTime.After(startTime) && localTime.Before(endTime), startTime, endTime, nil
}
