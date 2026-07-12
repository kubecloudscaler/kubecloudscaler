package cnpg

// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=clusters,verbs=get;list;update;patch

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

const (
	clusterKind    = "cnpgcluster"
	clusterGroup   = "postgresql.cnpg.io"
	clusterVersion = "v1"
)

func (c *Cnpg) init(client dynamic.Interface) {
	c.Client = client.Resource(schema.GroupVersionResource{
		Group:    clusterGroup,
		Version:  clusterVersion,
		Resource: "clusters",
	})
	c.AnnotationManager = utils.NewAnnotationManager()
}

// SetState sets the state of CloudNativePG Cluster resources based on the current period.
func (c *Cnpg) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	// Create adapters
	lister := &clusterLister{client: c.Client, logger: c.Logger}
	getter := &clusterGetter{client: c.Client}
	updater := &clusterUpdater{client: c.Client}

	// Create CNPG hibernation strategy
	strategy := base.NewCNPGHibernateStrategy(
		clusterKind,
		c.Logger,
		c.AnnotationManager,
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
