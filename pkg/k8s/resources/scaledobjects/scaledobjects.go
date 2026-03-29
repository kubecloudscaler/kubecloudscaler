package scaledobjects

// +kubebuilder:rbac:groups=keda.sh,resources=scaledobjects,verbs=get;list;update;patch

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

const (
	scaledObjectKind    = "scaledobject"
	scaledObjectGroup   = "keda.sh"
	scaledObjectVersion = "v1alpha1"
)

var scaledObjectGVK = schema.GroupVersionKind{
	Group:   scaledObjectGroup,
	Version: scaledObjectVersion,
	Kind:    "ScaledObject",
}

func (s *ScaledObjects) init(client dynamic.Interface) {
	s.Client = client.Resource(schema.GroupVersionResource{
		Group:    scaledObjectGroup,
		Version:  scaledObjectVersion,
		Resource: "scaledobjects",
	})
	s.AnnotationManager = utils.NewAnnotationManager()
}

// SetState sets the state of KEDA ScaledObject resources based on the current period.
func (s *ScaledObjects) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	// Create adapters
	lister := &scaledObjectLister{client: s.Client, logger: s.Logger}
	getter := &scaledObjectGetter{client: s.Client}
	updater := &scaledObjectUpdater{client: s.Client}

	// Create KEDA pause strategy
	strategy := base.NewKedaPauseStrategy(
		scaledObjectKind,
		getMinMaxReplicas,
		setMinMaxReplicas,
		s.Logger,
		s.AnnotationManager,
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
