package deployments

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;update;patch

import (
	"context"

	"k8s.io/client-go/kubernetes"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

func (d *Deployments) init(client kubernetes.Interface) {
	d.Client = client.AppsV1()
}

// SetState sets the state of Deployment resources based on the current period.
func (d *Deployments) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	// Create adapters
	lister := &deploymentLister{client: d.Client}
	getter := &deploymentGetter{client: d.Client}
	updater := &deploymentUpdater{client: d.Client}

	// Create annotation manager
	annotationMgr := utils.NewAnnotationManager()

	// Create scaling strategy
	strategy := base.NewIntReplicasStrategy(
		"deployment",
		getReplicas,
		setReplicas,
		d.Logger,
		annotationMgr,
	)

	// Create processor
	processor := base.NewProcessor(
		lister,
		getter,
		updater,
		strategy,
		d.Resource,
		d.Logger,
	)

	// Process resources
	return processor.ProcessResources(ctx)
}
