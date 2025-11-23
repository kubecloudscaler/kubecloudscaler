package ars

// +kubebuilder:rbac:groups=actions.github.com,resources=autoscalingrunnersets,verbs=get;list;update;patch

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/resources/base"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

const runnerSetKind = "autoscalingrunnerset"

func (h *GithubAutoscalingRunnersets) init(client dynamic.Interface) {
	h.Client = client.Resource(schema.GroupVersionResource{
		Group:    "actions.github.com",
		Version:  "v1alpha1",
		Resource: "autoscalingrunnersets",
	})
	h.AnnotationManager = utils.NewAnnotationManager()
}

// SetState sets the state of Github Autoscaling Runnersets resources based on the current period.
func (h *GithubAutoscalingRunnersets) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	// Create adapters
	lister := &runnerSetLister{client: h.Client}
	getter := &runnerSetGetter{client: h.Client}
	updater := &runnerSetUpdater{client: h.Client}

	// Create scaling strategy
	strategy := base.NewMinMaxReplicasStrategy(
		runnerSetKind,
		getMinMaxReplicas,
		setMinMaxReplicas,
		h.Logger,
		h.AnnotationManager,
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
