// Package utils provides utility functions for internal use in the kubecloudscaler project.
package utils

import (
	"github.com/rs/zerolog"
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

// noactionScalerPeriod is the constant definition for the noaction period.
// Defined once as a package-level variable to avoid re-allocation on every call.
var noactionScalerPeriod = &common.ScalerPeriod{
	Type: "noaction",
	Name: "noaction",
	Time: common.TimePeriod{
		Recurring: &common.RecurringPeriod{
			Days:      []string{"all"},
			StartTime: "00:00",
			EndTime:   "23:59",
			Once:      ptr.To(false),
		},
	},
}

// ValidatePeriod validates and returns the active period from the given periods.
// It checks if any period is currently active and updates the status accordingly.
// If forceRestore is true, it uses a restore period that spans the entire day.
func ValidatePeriod(
	logger *zerolog.Logger, periods []*common.ScalerPeriod,
	status *common.ScalerStatus, forceRestore bool,
) (*periodPkg.Period, error) {
	// check we are in an active period
	onPeriod, err := periodPkg.New(noactionScalerPeriod)
	if err != nil {
		logger.Error().Err(err).Msg("unable to load noaction period")

		return nil, ErrLoadNoactionPeriod
	}

	if !forceRestore {
		for _, period := range periods {
			logger.Debug().Msgf("checking period:\n  type => %s\n  def => %+v\n", period.Type, period.Time)
			curPeriod, err := periodPkg.New(period)
			if err != nil {
				logger.Error().Err(err).Msg("unable to load period")

				return nil, ErrLoadPeriod
			}

			logger.Debug().Msgf("is period:\n  type => %s\n  def => %v\n  isActive => %t", curPeriod.Type, curPeriod.Period, curPeriod.IsActive)

			logger.Debug().Msgf("starttime: %v\n", curPeriod.GetStartTime.String())
			logger.Debug().Msgf("endtime: %v\n", curPeriod.GetEndTime.String())

			if curPeriod.IsActive {
				onPeriod = curPeriod

				break
			}
		}
	}

	// if we are in a once period, we do nothing if the period has already been processed
	if ptr.Deref(onPeriod.Once, false) && status.CurrentPeriod != nil && status.CurrentPeriod.SpecSHA == onPeriod.Hash {
		logger.Debug().Msg("period already running")

		return onPeriod, ErrRunOncePeriod
	}

	// we always parse resources to scale or restore values
	logger.Debug().Msgf("is period:\n  type => %s\n  def => %v\n", onPeriod.Type, onPeriod.Period)

	// prepare status
	status.CurrentPeriod = &common.ScalerStatusPeriod{}
	status.CurrentPeriod.Spec = onPeriod.Period
	status.CurrentPeriod.SpecSHA = onPeriod.Hash
	status.CurrentPeriod.Type = onPeriod.Type
	status.CurrentPeriod.Name = onPeriod.Name

	return onPeriod, nil
}
