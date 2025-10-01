package hpa

import (
	"context"
	"fmt"

	autoscaleV2 "k8s.io/api/autoscaling/v2"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

func (h *HorizontalPodAutoscalers) init(client kubernetes.Interface) {
	h.Client = client.AutoscalingV2()
}

func (h *HorizontalPodAutoscalers) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	scalerStatusSuccess := []common.ScalerStatusSuccess{}
	scalerStatusFailed := []common.ScalerStatusFailed{}
	list := []autoscaleV2.HorizontalPodAutoscaler{}

	for _, ns := range h.Resource.NsList {
		h.Logger.Debug().Msgf("found namespace: %s", ns)

		deployList, err := h.Client.HorizontalPodAutoscalers(ns).List(ctx, h.Resource.ListOptions)
		if err != nil {
			h.Logger.Debug().Err(err).Msg("error listing hpas")

			return scalerStatusSuccess, scalerStatusFailed, fmt.Errorf("error listing hpas: %w", err)
		}

		list = append(list, deployList.Items...)
	}

	h.Logger.Debug().Msgf("number of hpas: %d", len(list))

	for _, dName := range list {
		h.Logger.Debug().Msgf("resource-name: %s", dName.Name)
		var deploy *autoscaleV2.HorizontalPodAutoscaler

		deploy, err := h.Client.HorizontalPodAutoscalers(dName.Namespace).Get(ctx, dName.Name, metaV1.GetOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "hpa",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		switch h.Resource.Period.Type {
		case "down":
			h.Logger.Debug().Msgf("scaling down: %s", dName.Name)

			deploy.Annotations = utils.AddMinMaxAnnotations(deploy.Annotations, h.Resource.Period, deploy.Spec.MinReplicas, deploy.Spec.MaxReplicas)
			deploy.Spec.MinReplicas = ptr.To(h.Resource.Period.MinReplicas)
			deploy.Spec.MaxReplicas = h.Resource.Period.MaxReplicas

		case "up":
			h.Logger.Debug().Msgf("scaling up: %s", dName.Name)

			deploy.Annotations = utils.AddMinMaxAnnotations(deploy.Annotations, h.Resource.Period, deploy.Spec.MinReplicas, deploy.Spec.MaxReplicas)
			deploy.Spec.MinReplicas = ptr.To(h.Resource.Period.MinReplicas)
			deploy.Spec.MaxReplicas = h.Resource.Period.MaxReplicas

		default:
			h.Logger.Debug().Msgf("restoring: %s", dName.Name)

			var isAlreadyRestored bool

			isAlreadyRestored, deploy.Spec.MinReplicas, deploy.Spec.MaxReplicas, deploy.Annotations, err = utils.RestoreMinMaxAnnotations(deploy.Annotations)
			if err != nil {
				scalerStatusFailed = append(
					scalerStatusFailed,
					common.ScalerStatusFailed{
						Kind:   "hpa",
						Name:   dName.Name,
						Reason: err.Error(),
					},
				)

				continue
			}

			if isAlreadyRestored {
				h.Logger.Debug().Msgf("nothing to do: %s", dName.Name)
				continue
			}
		}

		h.Logger.Debug().Msgf("update hpa: %s", dName.Name)

		_, err = h.
			Client.
			HorizontalPodAutoscalers(dName.Namespace).
			Update(ctx, deploy, metaV1.UpdateOptions{
				FieldManager: utils.FieldManager,
			})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "hpa",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		scalerStatusSuccess = append(
			scalerStatusSuccess,
			common.ScalerStatusSuccess{
				Kind: "hpa",
				Name: dName.Name,
			},
		)
	}

	return scalerStatusSuccess, scalerStatusFailed, nil
}

func (h *HorizontalPodAutoscalers) addAnnotations(hpa *autoscaleV2.HorizontalPodAutoscaler) {
	_, isExists := hpa.Annotations[utils.AnnotationsPrefix+"/original-minreplicas"]
	if !isExists {
		hpa.Annotations[utils.AnnotationsPrefix+"/original-minreplicas"] = fmt.Sprintf("%d", *hpa.Spec.MinReplicas)
	}

	_, isExists = hpa.Annotations[utils.AnnotationsPrefix+"/original-maxreplicas"]
	if !isExists {
		hpa.Annotations[utils.AnnotationsPrefix+"/original-maxreplicas"] = fmt.Sprintf("%d", hpa.Spec.MaxReplicas)
	}
}
