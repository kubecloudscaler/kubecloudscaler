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

package k8s

import (
	"context"
	"errors"
	"slices"
	"time"

	kubecloudscalerv1alpha1 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha1"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	k8sUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	k8sClient "github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils/client"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/resources"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ScalerReconciler reconciles a Scaler object
type ScalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=k8s,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=k8s/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kubecloudscaler.cloud,resources=k8s/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Scaler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// get the scaler object
	scaler := &kubecloudscalerv1alpha1.K8s{}
	if err := r.Get(ctx, req.NamespacedName, scaler); err != nil {
		log.Log.Error(err, "unable to fetch Scaler")

		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil
	}

	// get the k8s client in case of remote cluster
	kubeClient, err := k8sClient.GetClient()
	if err != nil {
		log.Log.Error(err, "unable to get k8s client")

		return ctrl.Result{RequeueAfter: utils.ReconcileErrorDuration}, nil
	}

	resourceConfig := resources.Config{
		K8s: &k8sUtils.Config{
			Client:                       kubeClient,
			Namespaces:                   scaler.Spec.Namespaces,
			ExcludeNamespaces:            scaler.Spec.ExcludeNamespaces,
			LabelSelector:                scaler.Spec.LabelSelector,
			ForceExcludeSystemNamespaces: scaler.Spec.ForceExcludeSystemNamespaces,
		},
	}

	resourceConfig.K8s.Period, err = utils.ValidatePeriod(scaler.Spec.Periods, &scaler.Status)
	if err != nil {
		if errors.Is(err, utils.ErrRunOncePeriod) {
			return ctrl.Result{RequeueAfter: time.Until(resourceConfig.K8s.Period.GetEndTime.Add(5 * time.Second))}, nil
		}

		scaler.Status.Comments = ptr.To(err.Error())

		if err := r.Status().Update(ctx, scaler); err != nil {
			log.Log.Error(err, "unable to update scaler status")
		}

		return ctrl.Result{Requeue: false}, nil
	}

	var (
		recSuccess []kubecloudscalerv1alpha1.ScalerStatusSuccess
		recFailed  []kubecloudscalerv1alpha1.ScalerStatusFailed
	)

	// filter resources type and execute the needed actions
	resourceList, err := r.validResourceList(ctx, scaler)
	if err != nil {
		log.Log.Error(err, "unable to get valid resources")
		scaler.Status.Comments = ptr.To(err.Error())

		if err := r.Status().Update(ctx, scaler); err != nil {
			log.Log.Error(err, "unable to update scaler status")
		}

		return ctrl.Result{}, nil
	}

	for _, resource := range resourceList {
		curResource, err := resources.NewResource(ctx, resource, resourceConfig)
		if err != nil {
			log.Log.Error(err, "unable to get resource")

			continue
		}

		success, failed, err := curResource.SetState(ctx)
		if err != nil {
			log.Log.Error(err, "unable to set resource state")

			continue
		}

		recSuccess = append(recSuccess, success...)
		recFailed = append(recFailed, failed...)
	}

	scaler.Status.CurrentPeriod.Successful = recSuccess
	scaler.Status.CurrentPeriod.Failed = recFailed
	scaler.Status.Comments = ptr.To("time period processed")

	if err := r.Status().Update(ctx, scaler); err != nil {
		log.Log.Error(err, "unable to update scaler status")
	}

	return ctrl.Result{RequeueAfter: utils.ReconcileSuccessDuration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubecloudscalerv1alpha1.K8s{}).
		WithEventFilter(utils.IgnoreDeletionPredicate()).
		Named("k8sScaler").
		Complete(r)
}

func (r *ScalerReconciler) validResourceList(ctx context.Context, scaler *kubecloudscalerv1alpha1.K8s) ([]string, error) {
	_ = log.FromContext(ctx)

	var (
		output []string
		isApp  bool
		isHpa  bool
	)

	// if no resources given, defaulting to deployments
	if len(scaler.Spec.Resources) == 0 {
		scaler.Spec.Resources = append(scaler.Spec.Resources, resources.DefaultResource)
	}

	for _, resource := range scaler.Spec.Resources {
		if !slices.Contains(scaler.Spec.ExcludeResources, resource) {
			if slices.Contains(utils.AppsResources, resource) {
				isApp = true
			}

			if slices.Contains(utils.HpaResources, resource) {
				isHpa = true
			}

			if isHpa && isApp {
				log.Log.V(1).Info(utils.ErrMixedAppsHPA.Error())

				return []string{}, utils.ErrMixedAppsHPA
			}

			output = append(output, resource)
		}
	}

	return output, nil
}

func (r *ScalerReconciler) cleanRef(ctx context.Context, scaler *kubecloudscalerv1alpha1.K8s) (*kubecloudscalerv1alpha1.K8s, error) {
	for i, period := range scaler.Spec.Periods {
		if period.Time.Recurring.StartTimeRef != nil {
			scalerKind, scalerName, PeriodName := periodPkg.ParseTimeRef(period.Time.Recurring.StartTimeRef)

			switch scalerKind {
			case "k8s":
				refScaler := &kubecloudscalerv1alpha1.K8s{}
				if err := r.Get(ctx,
					types.NamespacedName{
						Name:      scalerName,
						Namespace: "",
					},
					refScaler); err != nil {
					log.Log.Error(err, "unable to fetch Scaler")

					return &kubecloudscalerv1alpha1.K8s{}, err
				}
				for _, refPeriod := range refScaler.Spec.Periods {
					if refPeriod.Name == PeriodName {
						scaler.Spec.Periods[i].Time.Recurring.StartTime = refPeriod.Time.Recurring.StartTime
						break
					}
				}
			}
		}
	}
}
