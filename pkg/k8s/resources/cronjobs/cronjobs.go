// Package cronjobs provides CronJob scaling functionality for Kubernetes resources.
package cronjobs

import (
	"context"
	"fmt"

	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

func (c *Cronjobs) init(client kubernetes.Interface) {
	c.Client = client.BatchV1()
}

// SetState sets the state of CronJob resources based on the current period.
func (c *Cronjobs) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	scalerStatusSuccess := []common.ScalerStatusSuccess{}
	scalerStatusFailed := []common.ScalerStatusFailed{}
	list := []batchV1.CronJob{}

	// list all objects in all needed namespaces
	for _, ns := range c.Resource.NsList {
		c.Logger.Debug().Msgf("found namespace: %s", ns)

		cronList, err := c.Client.CronJobs(ns).List(ctx, c.Resource.ListOptions)
		if err != nil {
			c.Logger.Debug().Err(err).Msg("error listing cronjobs")

			return scalerStatusSuccess, scalerStatusFailed, fmt.Errorf("error listing cronjobs: %w", err)
		}

		list = append(list, cronList.Items...)
	}

	c.Logger.Debug().Msgf("number of cronjobs: %d", len(list))

	//nolint:gocritic // Range iteration of struct is acceptable, using index would reduce readability
	for _, cName := range list {
		c.Logger.Debug().Msgf("resource-name: %s", cName.Name)

		cronjob, err := c.Client.CronJobs(cName.Namespace).Get(ctx, cName.Name, metaV1.GetOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "cronjob",
					Name:   cName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		switch c.Resource.Period.Type {
		case "down":
			c.Logger.Debug().Msgf("scaling down: %s", cName.Name)

			cronjob.Annotations = utils.AddBoolAnnotations(cronjob.Annotations, c.Resource.Period, ptr.Deref(cronjob.Spec.Suspend, false))

			cronjob.Spec.Suspend = ptr.To(suspended)

		case "up":
			c.Logger.Debug().Msgf("scaling up: %s", cName.Name)

			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "cronjob",
					Name:   cName.Name,
					Reason: "cronjob can only be scaled down",
				},
			)

			continue

		default:
			c.Logger.Debug().Msgf("restoring: %s", cName.Name)

			var isAlreadyRestored bool

			isAlreadyRestored, cronjob.Spec.Suspend, cronjob.Annotations, err = utils.RestoreBoolAnnotations(cronjob.Annotations)
			if err != nil {
				scalerStatusFailed = append(
					scalerStatusFailed,
					common.ScalerStatusFailed{
						Kind:   "cronjob",
						Name:   cName.Name,
						Reason: err.Error(),
					},
				)

				continue
			}

			if isAlreadyRestored {
				c.Logger.Debug().Msgf("nothing to do: %s", cName.Name)
				continue
			}
		}

		c.Logger.Debug().Msgf("update cronjob: %s", cName.Name)

		_, err = c.Client.CronJobs(cName.Namespace).Update(ctx, cronjob, metaV1.UpdateOptions{
			FieldManager: utils.FieldManager,
		})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "cronjob",
					Name:   cName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		scalerStatusSuccess = append(
			scalerStatusSuccess,
			common.ScalerStatusSuccess{
				Kind: "cronjob",
				Name: cName.Name,
			},
		)
	}

	return scalerStatusSuccess, scalerStatusFailed, nil
}
