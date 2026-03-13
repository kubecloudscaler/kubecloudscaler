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
// It processes updates where the generation changes or where a DeletionTimestamp is first set,
// and deletes where the state is unknown.
func IgnoreDeletionPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Reconcile when spec changes (generation bump) or when deletion is initiated
			// (DeletionTimestamp transitions nil → non-nil) so finalizer cleanup runs immediately.
			generationChanged := e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
			deletionStarted := e.ObjectOld.GetDeletionTimestamp() == nil &&
				e.ObjectNew.GetDeletionTimestamp() != nil
			return generationChanged || deletionStarted
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return e.DeleteStateUnknown
		},
	}
}

// newNoactionPeriod returns a fresh ScalerPeriod representing "no active period".
// A factory is used instead of a package-level var to prevent callers from
// accidentally mutating shared state.
func newNoactionPeriod() *common.ScalerPeriod {
	return &common.ScalerPeriod{
		Type: common.PeriodType(periodPkg.NoactionPeriodName),
		Name: periodPkg.NoactionPeriodName,
		Time: common.TimePeriod{
			Recurring: &common.RecurringPeriod{
				Days:      []common.DayOfWeek{common.DayAll},
				StartTime: "00:00",
				EndTime:   "23:59",
				Once:      ptr.To(false),
			},
		},
	}
}

// SetActivePeriod determines the active period from the given list and updates status as a side effect.
// If forceRestore is true, the "noaction" period (spanning the entire day) is used unconditionally.
func SetActivePeriod(
	logger *zerolog.Logger, periods []*common.ScalerPeriod,
	status *common.ScalerStatus, forceRestore bool,
) (*periodPkg.Period, error) {
	// check we are in an active period
	onPeriod, err := periodPkg.New(newNoactionPeriod())
	if err != nil {
		logger.Error().Err(err).Msg("unable to load noaction period")

		return nil, ErrLoadNoactionPeriod
	}

	if !forceRestore {
		for _, period := range periods {
			curPeriod, err := periodPkg.New(period)
			if err != nil {
				logger.Error().Err(err).Msg("unable to load period")
				return nil, ErrLoadPeriod
			}
			logger.Debug().Str("period", string(period.Type)).Bool("active", curPeriod.IsActive).Msg("period checked")
			if curPeriod.IsActive {
				onPeriod = curPeriod
				break
			}
		}
	}

	if ptr.Deref(onPeriod.Once, false) && status.CurrentPeriod != nil && status.CurrentPeriod.SpecSHA == onPeriod.Hash {
		logger.Debug().Str("period", onPeriod.Name).Msg("run-once already applied")
		return onPeriod, ErrRunOncePeriod
	}

	// prepare status
	status.CurrentPeriod = &common.ScalerStatusPeriod{}
	status.CurrentPeriod.Spec = onPeriod.Spec
	status.CurrentPeriod.SpecSHA = onPeriod.Hash
	status.CurrentPeriod.Type = string(onPeriod.Type)
	status.CurrentPeriod.Name = onPeriod.Name

	return onPeriod, nil
}
