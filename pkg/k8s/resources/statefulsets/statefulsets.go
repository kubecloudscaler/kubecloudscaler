package statefulsets

import (
	"context"
	"fmt"

	kubecloudscalerv1alpha1 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha1"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	appsV1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Statefulsets struct {
	Resource *utils.K8sResource
	Client   v1.AppsV1Interface
}

func (d *Statefulsets) init(client *kubernetes.Clientset) {
	d.Client = client.AppsV1()
}

func (d *Statefulsets) SetState(ctx context.Context) ([]kubecloudscalerv1alpha1.ScalerStatusSuccess, []kubecloudscalerv1alpha1.ScalerStatusFailed, error) {
	_ = log.FromContext(ctx)
	scalerStatusSuccess := []kubecloudscalerv1alpha1.ScalerStatusSuccess{}
	scalerStatusFailed := []kubecloudscalerv1alpha1.ScalerStatusFailed{}
	list := []appsV1.StatefulSet{}

	for _, ns := range d.Resource.NsList {
		log.Log.V(1).Info("found namespace", "ns", ns)

		statefulList, err := d.Client.StatefulSets(ns).List(ctx, d.Resource.ListOptions)
		if err != nil {
			log.Log.V(1).Error(err, "error listing deployments")

			return scalerStatusSuccess, scalerStatusFailed, fmt.Errorf("error listing deployments: %w", err)
		}

		list = append(list, statefulList.Items...)
	}

	log.Log.V(1).Info("deployments", "number", len(list))

	for _, dName := range list {
		log.Log.V(1).Info("deployment", "name", dName.Name)
		var stateful *appsV1.StatefulSet

		stateful, err := d.Client.StatefulSets(dName.Namespace).Get(ctx, dName.Name, metaV1.GetOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				kubecloudscalerv1alpha1.ScalerStatusFailed{
					Kind:   "deployment",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		switch d.Resource.Period.Type {
		case "down":
			log.Log.V(1).Info("scaling down", "name", dName.Name)

			stateful.Annotations = utils.AddIntAnnotations(stateful.Annotations, d.Resource.Period, stateful.Spec.Replicas)

			stateful.Spec.Replicas = ptr.To(d.Resource.Period.MinReplicas)

		case "up":
			log.Log.V(1).Info("scaling up", "name", dName.Name)

			stateful.Annotations = utils.AddIntAnnotations(stateful.Annotations, d.Resource.Period, stateful.Spec.Replicas)
			stateful.Spec.Replicas = ptr.To(d.Resource.Period.MaxReplicas)

		default:
			log.Log.V(1).Info("restoring", "name", dName.Name)

			var isAlreadyRestored bool

			isAlreadyRestored, stateful.Spec.Replicas, stateful.Annotations, err = utils.RestoreIntAnnotations(stateful.Annotations)
			if err != nil {
				scalerStatusFailed = append(
					scalerStatusFailed,
					kubecloudscalerv1alpha1.ScalerStatusFailed{
						Kind:   "deployment",
						Name:   dName.Name,
						Reason: err.Error(),
					},
				)

				continue
			}

			if isAlreadyRestored {
				log.Log.V(1).Info("nothing to do", "name", dName.Name)
				continue
			}
		}

		log.Log.V(1).Info("update deployment", "name", dName.Name)

		_, err = d.Client.StatefulSets(dName.Namespace).Update(ctx, stateful, metaV1.UpdateOptions{
			FieldManager: utils.FieldManager,
		})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				kubecloudscalerv1alpha1.ScalerStatusFailed{
					Kind:   "deployment",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		scalerStatusSuccess = append(
			scalerStatusSuccess,
			kubecloudscalerv1alpha1.ScalerStatusSuccess{
				Kind: "deployment",
				Name: dName.Name,
			},
		)
	}

	return scalerStatusSuccess, scalerStatusFailed, nil
}
