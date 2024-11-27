package hpa

import (
	"context"
	"fmt"

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
	isAlreadyRestored := false

	for _, ns := range d.Resource.NsList {
		log.Log.V(1).Info("found namespace", "ns", ns)

		deployList, err := d.Client.HorizontalPodAutoscalers(ns).List(ctx, d.Resource.ListOptions)
		if err != nil {
			log.Log.V(1).Error(err, "error listing hpas")

			return scalerStatusSuccess, scalerStatusFailed, err
		}

		list = append(list, deployList.Items...)
	}

	log.Log.V(1).Info("hpas", "number", len(list))

	for _, dName := range list {
		log.Log.V(1).Info("hpa", "name", dName.Name)
		var deploy *autoscaleV2.HorizontalPodAutoscaler

		deploy, err := d.Client.HorizontalPodAutoscalers(dName.Namespace).Get(ctx, dName.Name, metaV1.GetOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				cloudscaleriov1alpha1.ScalerStatusFailed{
					Kind:   "hpa",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		switch d.Resource.Period.Type {
		case "down":
			log.Log.V(1).Info("scaling down", "name", dName.Name)

			deploy.Annotations = utils.AddMinMaxAnnotations(deploy.Annotations, d.Resource.Period, deploy.Spec.MinReplicas, deploy.Spec.MaxReplicas)
			deploy.Spec.MinReplicas = ptr.To(d.Resource.Period.MinReplicas)
			deploy.Spec.MaxReplicas = d.Resource.Period.MaxReplicas

		case "up":
			log.Log.V(1).Info("scaling up", "name", dName.Name)

			deploy.Annotations = utils.AddMinMaxAnnotations(deploy.Annotations, d.Resource.Period, deploy.Spec.MinReplicas, deploy.Spec.MaxReplicas)
			deploy.Spec.MinReplicas = ptr.To(d.Resource.Period.MinReplicas)
			deploy.Spec.MaxReplicas = d.Resource.Period.MaxReplicas

		default:
			log.Log.V(1).Info("restoring", "name", dName.Name)

			isAlreadyRestored, deploy.Spec.MinReplicas, deploy.Spec.MaxReplicas, deploy.Annotations, err = utils.RestoreMinMaxAnnotations(deploy.Annotations)
			if err != nil {
				scalerStatusFailed = append(
					scalerStatusFailed,
					cloudscaleriov1alpha1.ScalerStatusFailed{
						Kind:   "hpa",
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

		log.Log.V(1).Info("update hpa", "name", dName.Name)

		_, err = d.
			Client.
			HorizontalPodAutoscalers(dName.Namespace).
			Update(ctx, deploy, metaV1.UpdateOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				cloudscaleriov1alpha1.ScalerStatusFailed{
					Kind:   "hpa",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		scalerStatusSuccess = append(
			scalerStatusSuccess,
			cloudscaleriov1alpha1.ScalerStatusSuccess{
				Kind: "hpa",
				Name: dName.Name,
			},
		)
	}

	return scalerStatusSuccess, scalerStatusFailed, nil
}

func (d *HorizontalPodAutoscalers) addAnnotations(deploy *autoscaleV2.HorizontalPodAutoscaler) {
	// deploy.Annotations = utils.AddAnnotations(deploy.Annotations, d.Resource.Period)

	_, isExists := deploy.Annotations[utils.AnnotationsPrefix+"/original-minreplicas"]
	if !isExists {
		deploy.Annotations[utils.AnnotationsPrefix+"/original-minreplicas"] = fmt.Sprintf("%d", *deploy.Spec.MinReplicas)
	}

	_, isExists = deploy.Annotations[utils.AnnotationsPrefix+"/original-maxreplicas"]
	if !isExists {
		deploy.Annotations[utils.AnnotationsPrefix+"/original-maxreplicas"] = fmt.Sprintf("%d", deploy.Spec.MaxReplicas)
	}
}
