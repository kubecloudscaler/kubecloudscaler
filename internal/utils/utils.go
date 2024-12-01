package utils

import (
	"fmt"

	k8scloudscalerv1alpha1 "github.com/k8scloudscaler/k8scloudscaler/api/v1alpha1"
	periodPkg "github.com/k8scloudscaler/k8scloudscaler/pkg/period"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func IgnoreDeletionPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}
}

func ValidatePeriod(periods []*k8scloudscalerv1alpha1.ScalerPeriod, status *k8scloudscalerv1alpha1.ScalerStatus) (*periodPkg.Period, error) {
	// check we are in an active period
	restorePeriod := &k8scloudscalerv1alpha1.ScalerPeriod{
		Type: "restore",
		Time: k8scloudscalerv1alpha1.TimePeriod{
			Recurring: &k8scloudscalerv1alpha1.RecurringPeriod{
				Days:      []string{"all"},
				StartTime: "00:00",
				EndTime:   "00:00",
				Once:      ptr.To(false),
			},
		},
	}

	onPeriod, err := periodPkg.New(restorePeriod)
	if err != nil {
		log.Log.Error(err, "unable to load restore period")

		return nil, ErrLoadRestorePeriod
	}

	for _, period := range periods {
		log.Log.V(1).Info(fmt.Sprintf("checking period:\n  type => %s\n  def => %+v\n", period.Type, period.Time))
		curPeriod, err := periodPkg.New(period)
		if err != nil {
			log.Log.Error(err, "unable to load period")

			return nil, ErrLoadPeriod
		}

		log.Log.V(1).Info(fmt.Sprintf("is period:\n  type => %s\n  def => %v\n  isActive => %t", curPeriod.Type, curPeriod.Period, curPeriod.IsActive))

		log.Log.V(1).Info(fmt.Sprintf("starttime: %v\n", curPeriod.GetStartTime.String()))
		log.Log.V(1).Info(fmt.Sprintf("endtime: %v\n", curPeriod.GetEndTime.String()))

		if curPeriod.IsActive {
			onPeriod = curPeriod

			break
		}
	}

	// if we are in a once period, we do nothing if the period has already been processed
	if ptr.Deref(onPeriod.Once, false) && status.CurrentPeriod.SpecSHA == onPeriod.Hash {
		log.Log.V(1).Info("period already running")

		return onPeriod, ErrRunOncePeriod
	}

	// we always parse resources to scale or restore values
	log.Log.V(1).Info(fmt.Sprintf("is period:\n  type => %s\n  def => %v\n", onPeriod.Type, onPeriod.Period))

	// prepare status
	status.CurrentPeriod = &k8scloudscalerv1alpha1.ScalerStatusPeriod{}
	status.CurrentPeriod.Spec = onPeriod.Period
	status.CurrentPeriod.SpecSHA = onPeriod.Hash

	return onPeriod, nil
}
