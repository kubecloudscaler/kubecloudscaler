package deployments

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cloudscalerio/cloudscaler/api/common"
	"github.com/cloudscalerio/cloudscaler/pkg/k8s/utils"
	appsV1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Deployments struct {
	Resource *utils.K8sResource
	Client   v1.AppsV1Interface
}

func (d *Deployments) init(client *kubernetes.Clientset) {
	d.Client = client.AppsV1()
}

func (d *Deployments) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	_ = log.FromContext(ctx)
	scalerStatusSuccess := []common.ScalerStatusSuccess{}
	scalerStatusFailed := []common.ScalerStatusFailed{}
	list := []appsV1.Deployment{}

	for _, ns := range d.Resource.NsList {
		log.Log.V(1).Info("found namespace", "ns", ns)

		deployList, err := d.Client.Deployments(ns).List(ctx, d.Resource.ListOptions)
		if err != nil {
			log.Log.V(1).Error(err, "error listing deployments")

			return scalerStatusSuccess, scalerStatusFailed, err
		}

		list = append(list, deployList.Items...)
	}

	log.Log.V(1).Info("deployments", "number", len(list))

	for _, dName := range list {
		log.Log.V(1).Info("deployment", "name", dName.Name)
		var deploy *appsV1.Deployment

		deploy, err := d.Client.Deployments(dName.Namespace).Get(ctx, dName.Name, metaV1.GetOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "deployment",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		switch d.Resource.Period.Period.Type {
		case "down":
			log.Log.V(1).Info("scaling down", "name", dName.Name)

			d.addAnnotations(deploy)

			deploy.Spec.Replicas = d.Resource.Period.Period.MinReplicas

		case "up":
			log.Log.V(1).Info("scaling up", "name", dName.Name)

			d.addAnnotations(deploy)

			deploy.Spec.Replicas = d.Resource.Period.Period.MaxReplicas

		case "restore":
			log.Log.V(1).Info("restoring", "name", dName.Name)

			err := d.restore(deploy)
			if err != nil {
				scalerStatusFailed = append(
					scalerStatusFailed,
					common.ScalerStatusFailed{
						Kind:   "deployment",
						Name:   dName.Name,
						Reason: err.Error(),
					},
				)

				continue
			}
		default:
			log.Log.V(1).Info("unknown period type", "type", d.Resource.Period.Period.Type) // case "nominal":
		}

		log.Log.V(1).Info("update deployment", "name", dName.Name)

		_, err = d.Client.Deployments(dName.Namespace).Update(ctx, deploy, metaV1.UpdateOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "deployment",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		scalerStatusSuccess = append(
			scalerStatusSuccess,
			common.ScalerStatusSuccess{
				Kind: "deployment",
				Name: dName.Name,
			},
		)
	}

	return scalerStatusSuccess, scalerStatusFailed, nil
}

func (d *Deployments) addAnnotations(deploy *appsV1.Deployment) {
	deploy.Annotations = utils.AddAnnotations(deploy.Annotations, d.Resource.Period.Period)

	_, isExists := deploy.Annotations[utils.AnnotationsPrefix+"/original-replicas"]
	if !isExists {
		deploy.Annotations[utils.AnnotationsPrefix+"/original-replicas"] = fmt.Sprintf("%d", *deploy.Spec.Replicas)
	}
}

func (d *Deployments) restore(deploy *appsV1.Deployment) error {
	rep, isExists := deploy.Annotations[utils.AnnotationsPrefix+"/original-replicas"]
	if isExists {
		repAsInt, err := strconv.Atoi(rep)
		if err != nil {
			return err
		}

		deploy.Spec.Replicas = ptr.To(int32(repAsInt))
	}

	deploy.Annotations = utils.RemoveAnnotations(deploy.Annotations)

	return nil
}
