package cronjobs

import (
	"context"
	"strconv"

	"github.com/cloudscalerio/cloudscaler/api/common"
	"github.com/cloudscalerio/cloudscaler/pkg/k8s/utils"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Cronjobs struct {
	Resource *utils.K8sResource
	Client   v1.BatchV1Interface
}

func (c *Cronjobs) init(client *kubernetes.Clientset) {
	c.Client = client.BatchV1()
}

func (c *Cronjobs) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	_ = log.FromContext(ctx)
	scalerStatusSuccess := []common.ScalerStatusSuccess{}
	scalerStatusFailed := []common.ScalerStatusFailed{}
	list := []batchV1.CronJob{}

	for _, ns := range c.Resource.NsList {
		log.Log.V(1).Info("found namespace", "ns", ns)

		cronList, err := c.Client.CronJobs(ns).List(ctx, c.Resource.ListOptions)
		if err != nil {
			log.Log.V(1).Error(err, "error listing deployments")

			return scalerStatusSuccess, scalerStatusFailed, err
		}

		list = append(list, cronList.Items...)
	}

	log.Log.V(1).Info("cronjobs", "number", len(list))

	for _, cName := range list {
		log.Log.V(1).Info("cronjobs", "name", cName.Name)

		cronjob, err := c.Client.CronJobs(cName.Namespace).Get(ctx, cName.Name, metaV1.GetOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "deployment",
					Name:   cName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		switch c.Resource.Period.Period.Type {
		case "down":
			log.Log.V(1).Info("scaling down", "name", cName.Name)

			c.addAnnotations(cronjob)

			cronjob.Spec.Suspend = ptr.To(suspended)

		case "up":
			log.Log.V(1).Info("scaling up", "name", cName.Name)

			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "cronjob",
					Name:   cName.Name,
					Reason: "cronjob can only be scaled down",
				},
			)

			continue

		case "restore":
			log.Log.V(1).Info("restoring", "name", cName.Name)

			err := c.restore(cronjob)
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
		default:
			log.Log.V(1).Info("unknown period type", "type", c.Resource.Period.Period.Type) // case "nominal":
		}

		log.Log.V(1).Info("update cronjob", "name", cName.Name)

		_, err = c.Client.CronJobs(cName.Namespace).Update(ctx, cronjob, metaV1.UpdateOptions{})
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

func (c *Cronjobs) addAnnotations(cronjob *batchV1.CronJob) {
	if cronjob.Annotations == nil {
		cronjob.Annotations = map[string]string{}
	}

	cronjob.Annotations = utils.AddAnnotations(cronjob.Annotations, c.Resource.Period.Period)

	_, isExists := cronjob.Annotations[utils.AnnotationsPrefix+"/suspended"]
	if !isExists {
		cronjob.Annotations[utils.AnnotationsPrefix+"/suspended"] = strconv.FormatBool(*cronjob.Spec.Suspend)
	}
}

func (c *Cronjobs) restore(cronjob *batchV1.CronJob) error {
	rep, isExists := cronjob.Annotations[utils.AnnotationsPrefix+"/suspended"]
	if isExists {
		repAsBool, err := strconv.ParseBool(rep)
		if err != nil {
			return err
		}

		cronjob.Spec.Suspend = ptr.To(repAsBool)
	}

	cronjob.Annotations = utils.RemoveAnnotations(cronjob.Annotations)

	return nil
}
