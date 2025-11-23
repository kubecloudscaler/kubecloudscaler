package statefulsets

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;update;patch

import (
	"context"

	"k8s.io/client-go/kubernetes"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

func (s *Statefulsets) init(client kubernetes.Interface) {
	s.Client = client.AppsV1()
}

// SetState sets the state of StatefulSet resources based on the current period.
func (s *Statefulsets) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	// Create adapters
	lister := &statefulSetLister{client: s.Client}
	getter := &statefulSetGetter{client: s.Client}
	updater := &statefulSetUpdater{client: s.Client}

	// Create annotation manager
	annotationMgr := utils.NewAnnotationManager()

	// Create scaling strategy
	strategy := base.NewIntReplicasStrategy(
		"statefulset",
		getReplicas,
		setReplicas,
		s.Logger,
		annotationMgr,
	)

	// Create processor
	processor := base.NewProcessor(
		lister,
		getter,
		updater,
		strategy,
		s.Resource,
		s.Logger,
	)

	// Process resources
	return processor.ProcessResources(ctx)
}
