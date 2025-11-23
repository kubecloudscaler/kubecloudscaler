// Package cronjobs provides CronJob scaling functionality for Kubernetes resources.
package cronjobs

// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;update;patch

import (
	"context"

	"k8s.io/client-go/kubernetes"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

func (c *Cronjobs) init(client kubernetes.Interface) {
	c.Client = client.BatchV1()
}

// SetState sets the state of CronJob resources based on the current period.
func (c *Cronjobs) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	// Create adapters
	lister := &cronJobLister{client: c.Client}
	getter := &cronJobGetter{client: c.Client}
	updater := &cronJobUpdater{client: c.Client}

	// Create annotation manager
	annotationMgr := utils.NewAnnotationManager()

	// Create scaling strategy
	strategy := base.NewBoolSuspendStrategy(
		"cronjob",
		getSuspend,
		setSuspend,
		suspended,
		c.Logger,
		onUpError,
		annotationMgr,
	)

	// Create processor
	processor := base.NewProcessor(
		lister,
		getter,
		updater,
		strategy,
		c.Resource,
		c.Logger,
	)

	// Process resources
	return processor.ProcessResources(ctx)
}
