package hpa

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cloudscalerio/cloudscaler/api/common"
	"github.com/cloudscalerio/cloudscaler/pkg/k8s/utils"
	autoscaleV2 "k8s.io/api/autoscaling/v2"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type HorizontalPodAutoscalers struct {
	Resource *utils.K8sResource
}

func (d *HorizontalPodAutoscalers) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	_ = log.FromContext(ctx)
	scalerStatusSuccess := []common.ScalerStatusSuccess{}
	scalerStatusFailed := []common.ScalerStatusFailed{}
	list := []autoscaleV2.HorizontalPodAutoscaler{}

	for _, ns := range d.Resource.NsList {
		log.Log.V(1).Info("found namespace", "ns", ns)

		deployList, err := d.Resource.Config.Client.AutoscalingV2().HorizontalPodAutoscalers(ns).List(ctx, d.Resource.ListOptions)
		if err != nil {
			log.Log.V(1).Error(err, "error listing deployments")

			return scalerStatusSuccess, scalerStatusFailed, err
		}

		list = append(list, deployList.Items...)
	}

	log.Log.V(1).Info("deployments", "number", len(list))

	for _, dName := range list {
		log.Log.V(1).Info("deployment", "name", dName.Name)
		var deploy *autoscaleV2.HorizontalPodAutoscaler

		deploy, err := d.Resource.Config.Client.AutoscalingV2().HorizontalPodAutoscalers(dName.Namespace).Get(ctx, dName.Name, metaV1.GetOptions{})
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

		switch d.Resource.Config.Period.Period.Type {
		case "down":
			log.Log.V(1).Info("scaling down", "name", dName.Name)

			d.addAnnotations(deploy)

			deploy.Spec.MinReplicas = d.Resource.Config.Period.Period.MinReplicas

		case "up":
			log.Log.V(1).Info("scaling up", "name", dName.Name)

			d.addAnnotations(deploy)

			deploy.Spec.MinReplicas = d.Resource.Config.Period.Period.MinReplicas
			deploy.Spec.MaxReplicas = *d.Resource.Config.Period.Period.MaxReplicas

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
			log.Log.V(1).Info("unknown period type", "type", d.Resource.Config.Period.Period.Type) // case "nominal":
		}

		log.Log.V(1).Info("update deployment", "name", dName.Name)

		_, err = d.
			Resource.
			Config.
			Client.
			AutoscalingV2().
			HorizontalPodAutoscalers(dName.Namespace).
			Update(ctx, deploy, metaV1.UpdateOptions{})
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

func (d *HorizontalPodAutoscalers) addAnnotations(deploy *autoscaleV2.HorizontalPodAutoscaler) {
	deploy.Annotations = utils.AddAnnotations(deploy.Annotations, d.Resource.Config.Period.Period)

	_, isExists := deploy.Annotations[utils.AnnotationsPrefix+"/original-minreplicas"]
	if !isExists {
		deploy.Annotations[utils.AnnotationsPrefix+"/original-minreplicas"] = fmt.Sprintf("%d", *deploy.Spec.MinReplicas)
	}

	_, isExists = deploy.Annotations[utils.AnnotationsPrefix+"/original-maxreplicas"]
	if !isExists {
		deploy.Annotations[utils.AnnotationsPrefix+"/original-maxreplicas"] = fmt.Sprintf("%d", deploy.Spec.MaxReplicas)
	}
}

func (d *HorizontalPodAutoscalers) restore(deploy *autoscaleV2.HorizontalPodAutoscaler) error {
	rep, isExists := deploy.Annotations[utils.AnnotationsPrefix+"/original-minreplicas"]
	if isExists {
		repAsInt, err := strconv.Atoi(rep)
		if err != nil {
			return err
		}

		deploy.Spec.MinReplicas = ptr.To(int32(repAsInt))
	}

	rep, isExists = deploy.Annotations[utils.AnnotationsPrefix+"/original-maxreplicas"]
	if isExists {
		repAsInt, err := strconv.Atoi(rep)
		if err != nil {
			return err
		}

		deploy.Spec.MaxReplicas = int32(repAsInt)
	}

	deploy.Annotations = utils.RemoveAnnotations(deploy.Annotations)

	return nil
}
