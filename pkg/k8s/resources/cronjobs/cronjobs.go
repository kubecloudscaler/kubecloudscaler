package cronjobs

import (
	"context"

	cloudscaleriov1alpha1 "github.com/cloudscalerio/cloudscaler/api/v1alpha1"
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

func (c *Cronjobs) SetState(ctx context.Context) ([]cloudscaleriov1alpha1.ScalerStatusSuccess, []cloudscaleriov1alpha1.ScalerStatusFailed, error) {
	_ = log.FromContext(ctx)
	scalerStatusSuccess := []cloudscaleriov1alpha1.ScalerStatusSuccess{}
	scalerStatusFailed := []cloudscaleriov1alpha1.ScalerStatusFailed{}
	list := []batchV1.CronJob{}

	// list all objects in all needed namespaces
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
				cloudscaleriov1alpha1.ScalerStatusFailed{
					Kind:   "deployment",
					Name:   cName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		switch c.Resource.Period.Type {
		case "down":
			log.Log.V(1).Info("scaling down", "name", cName.Name)

			cronjob.Annotations = utils.AddBoolAnnotations(cronjob.Annotations, c.Resource.Period, suspended)

			cronjob.Spec.Suspend = ptr.To(suspended)

		case "up":
			log.Log.V(1).Info("scaling up", "name", cName.Name)

			scalerStatusFailed = append(
				scalerStatusFailed,
				cloudscaleriov1alpha1.ScalerStatusFailed{
					Kind:   "cronjob",
					Name:   cName.Name,
					Reason: "cronjob can only be scaled down",
				},
			)

			continue

		default:
			log.Log.V(1).Info("restoring", "name", cName.Name)

			cronjob.Spec.Suspend, cronjob.Annotations, err = utils.RestoreBool(cronjob.Annotations)
			if err != nil {
				scalerStatusFailed = append(
					scalerStatusFailed,
					cloudscaleriov1alpha1.ScalerStatusFailed{
						Kind:   "cronjob",
						Name:   cName.Name,
						Reason: err.Error(),
					},
				)

				continue
			}
		}

		log.Log.V(1).Info("update cronjob", "name", cName.Name)

		_, err = c.Client.CronJobs(cName.Namespace).Update(ctx, cronjob, metaV1.UpdateOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				cloudscaleriov1alpha1.ScalerStatusFailed{
					Kind:   "cronjob",
					Name:   cName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		scalerStatusSuccess = append(
			scalerStatusSuccess,
			cloudscaleriov1alpha1.ScalerStatusSuccess{
				Kind: "cronjob",
				Name: cName.Name,
			},
		)
	}

	return scalerStatusSuccess, scalerStatusFailed, nil
}
