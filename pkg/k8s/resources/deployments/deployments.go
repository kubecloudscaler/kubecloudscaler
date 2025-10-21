package deployments

import (
	"context"
	"fmt"

	appsV1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

func (d *Deployments) init(client kubernetes.Interface) {
	d.Client = client.AppsV1()
}

// SetState sets the state of Deployment resources based on the current period.
func (d *Deployments) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	scalerStatusSuccess := []common.ScalerStatusSuccess{}
	scalerStatusFailed := []common.ScalerStatusFailed{}
	list := []appsV1.Deployment{}

	for _, ns := range d.Resource.NsList {
		d.Logger.Debug().Msgf("found namespace: %s", ns)

		deployList, err := d.Client.Deployments(ns).List(ctx, d.Resource.ListOptions)
		if err != nil {
			d.Logger.Debug().Err(err).Msg("error listing deployments")

			return scalerStatusSuccess, scalerStatusFailed, fmt.Errorf("error listing deployments: %w", err)
		}

		list = append(list, deployList.Items...)
	}

	d.Logger.Debug().Msgf("number of deployments: %d", len(list))

	for _, dName := range list {
		d.Logger.Debug().Msgf("resource-name: %s", dName.Name)
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

		switch d.Resource.Period.Type {
		case "down":
			d.Logger.Debug().Msgf("scaling down: %s", dName.Name)

			deploy.Annotations = utils.AddIntAnnotations(deploy.Annotations, d.Resource.Period, deploy.Spec.Replicas)
			deploy.Spec.Replicas = ptr.To(d.Resource.Period.MinReplicas)

		case "up":
			d.Logger.Debug().Msgf("scaling up: %s", dName.Name)

			deploy.Annotations = utils.AddIntAnnotations(deploy.Annotations, d.Resource.Period, deploy.Spec.Replicas)
			deploy.Spec.Replicas = ptr.To(d.Resource.Period.MaxReplicas)

		default:
			d.Logger.Debug().Msgf("restoring: %s", dName.Name)

			var isAlreadyRestored bool

			isAlreadyRestored, deploy.Spec.Replicas, deploy.Annotations, err = utils.RestoreIntAnnotations(deploy.Annotations)
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

			if isAlreadyRestored {
				d.Logger.Debug().Msgf("nothing to do: %s", dName.Name)
				continue
			}
		}

		d.Logger.Debug().Msgf("update deployment: %s", dName.Name)

		_, err = d.Client.Deployments(dName.Namespace).Update(ctx, deploy, metaV1.UpdateOptions{
			FieldManager: utils.FieldManager,
		})
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
