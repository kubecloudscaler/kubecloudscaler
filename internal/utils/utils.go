// Package utils provides utility functions for internal use in the kubecloudscaler project.
package utils

import (
	"github.com/rs/zerolog/log"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// IgnoreDeletionPredicate returns a predicate that ignores deletion events.
// It only processes updates where the generation changes and deletes where the state is unknown.
func IgnoreDeletionPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return e.DeleteStateUnknown
		},
	}
}

// ValidatePeriod validates and returns the active period from the given periods.
// It checks if any period is currently active and updates the status accordingly.
// If forceRestore is true, it uses a restore period that spans the entire day.
func ValidatePeriod(periods []*common.ScalerPeriod, status *common.ScalerStatus, forceRestore bool) (*periodPkg.Period, error) {
	// check we are in an active period
	restorePeriod := &common.ScalerPeriod{
		Type: "restore",
		Time: common.TimePeriod{
			Recurring: &common.RecurringPeriod{
				Days:      []string{"all"},
				StartTime: "00:00",
				EndTime:   "23:59",
				Once:      ptr.To(false),
			},
		},
	}

	onPeriod, err := periodPkg.New(restorePeriod)
	if err != nil {
		log.Error().Err(err).Msg("unable to load restore period")

		return nil, ErrLoadRestorePeriod
	}

	if !forceRestore {
		for _, period := range periods {
			log.Debug().Msgf("checking period:\n  type => %s\n  def => %+v\n", period.Type, period.Time)
			curPeriod, err := periodPkg.New(period)
			if err != nil {
				log.Error().Err(err).Msg("unable to load period")

				return nil, ErrLoadPeriod
			}

			log.Debug().Msgf("is period:\n  type => %s\n  def => %v\n  isActive => %t", curPeriod.Type, curPeriod.Period, curPeriod.IsActive)

			log.Debug().Msgf("starttime: %v\n", curPeriod.GetStartTime.String())
			log.Debug().Msgf("endtime: %v\n", curPeriod.GetEndTime.String())

			if curPeriod.IsActive {
				onPeriod = curPeriod

				break
			}
		}
	}

	// if we are in a once period, we do nothing if the period has already been processed
	if ptr.Deref(onPeriod.Once, false) && status.CurrentPeriod.SpecSHA == onPeriod.Hash {
		log.Debug().Msg("period already running")

		return onPeriod, ErrRunOncePeriod
	}

	// we always parse resources to scale or restore values
	log.Debug().Msgf("is period:\n  type => %s\n  def => %v\n", onPeriod.Type, onPeriod.Period)

	// prepare status
	status.CurrentPeriod = &common.ScalerStatusPeriod{}
	status.CurrentPeriod.Spec = onPeriod.Period
	status.CurrentPeriod.SpecSHA = onPeriod.Hash
	status.CurrentPeriod.Type = onPeriod.Type

	return onPeriod, nil
}
