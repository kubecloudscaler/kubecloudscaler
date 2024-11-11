package hpa

import (
	"context"
	"fmt"
	"strconv"

	cloudscaleriov1alpha1 "github.com/cloudscalerio/cloudscaler/api/v1alpha1"
	"github.com/cloudscalerio/cloudscaler/pkg/k8s/utils"
	autoscaleV2 "k8s.io/api/autoscaling/v2"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type HorizontalPodAutoscalers struct {
	Resource *utils.K8sResource
	Client   v2.AutoscalingV2Interface
}

func (d *HorizontalPodAutoscalers) Init(client *kubernetes.Clientset) {
	d.Client = client.AutoscalingV2()
}

func (d *HorizontalPodAutoscalers) SetState(ctx context.Context) ([]cloudscaleriov1alpha1.ScalerStatusSuccess, []cloudscaleriov1alpha1.ScalerStatusFailed, error) {
	_ = log.FromContext(ctx)
	scalerStatusSuccess := []cloudscaleriov1alpha1.ScalerStatusSuccess{}
	scalerStatusFailed := []cloudscaleriov1alpha1.ScalerStatusFailed{}
	list := []autoscaleV2.HorizontalPodAutoscaler{}

	for _, ns := range d.Resource.NsList {
		log.Log.V(1).Info("found namespace", "ns", ns)

		deployList, err := d.Client.HorizontalPodAutoscalers(ns).List(ctx, d.Resource.ListOptions)
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

		deploy, err := d.Client.HorizontalPodAutoscalers(dName.Namespace).Get(ctx, dName.Name, metaV1.GetOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				cloudscaleriov1alpha1.ScalerStatusFailed{
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

			d.addAnnotations(deploy)

			deploy.Spec.MinReplicas = d.Resource.Period.MinReplicas

		case "up":
			log.Log.V(1).Info("scaling up", "name", dName.Name)

			d.addAnnotations(deploy)

			deploy.Spec.MinReplicas = d.Resource.Period.MinReplicas
			deploy.Spec.MaxReplicas = *d.Resource.Period.MaxReplicas

		case "restore":
			log.Log.V(1).Info("restoring", "name", dName.Name)

			err := d.restore(deploy)
			if err != nil {
				scalerStatusFailed = append(
					scalerStatusFailed,
					cloudscaleriov1alpha1.ScalerStatusFailed{
						Kind:   "deployment",
						Name:   dName.Name,
						Reason: err.Error(),
					},
				)

				continue
			}
		default:
			log.Log.V(1).Info("unknown period type", "type", d.Resource.Period.Type) // case "nominal":
		}

		log.Log.V(1).Info("update deployment", "name", dName.Name)

		_, err = d.
			Client.
			HorizontalPodAutoscalers(dName.Namespace).
			Update(ctx, deploy, metaV1.UpdateOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				cloudscaleriov1alpha1.ScalerStatusFailed{
					Kind:   "deployment",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		scalerStatusSuccess = append(
			scalerStatusSuccess,
			cloudscaleriov1alpha1.ScalerStatusSuccess{
				Kind: "deployment",
				Name: dName.Name,
			},
		)
	}

	return scalerStatusSuccess, scalerStatusFailed, nil
}

func (d *HorizontalPodAutoscalers) addAnnotations(deploy *autoscaleV2.HorizontalPodAutoscaler) {
	deploy.Annotations = utils.AddAnnotations(deploy.Annotations, d.Resource.Period)

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
