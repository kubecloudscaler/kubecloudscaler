package hpa

// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;update;patch

import (
	"context"

	"k8s.io/client-go/kubernetes"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

func (h *HorizontalPodAutoscalers) init(client kubernetes.Interface) {
	h.Client = client.AutoscalingV2()
}

// SetState sets the state of HorizontalPodAutoscaler resources based on the current period.
func (h *HorizontalPodAutoscalers) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	// Create adapters
	lister := &hpaLister{client: h.Client}
	getter := &hpaGetter{client: h.Client}
	updater := &hpaUpdater{client: h.Client}

	// Create annotation manager
	annotationMgr := utils.NewAnnotationManager()

	// Create scaling strategy
	strategy := base.NewMinMaxReplicasStrategy(
		"hpa",
		getMinMaxReplicas,
		setMinMaxReplicas,
		h.Logger,
		annotationMgr,
	)

	// Create processor
	processor := base.NewProcessor(
		lister,
		getter,
		updater,
		strategy,
		h.Resource,
		h.Logger,
	)

	// Process resources
	return processor.ProcessResources(ctx)
}
