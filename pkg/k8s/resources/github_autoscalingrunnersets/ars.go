package ars

// +kubebuilder:rbac:groups=actions.github.com,resources=autoscalingrunnersets,verbs=get;list;update;patch

import (
	"context"
	"fmt"

	actionsV1alpha1 "github.com/actions/actions-runner-controller/apis/actions.github.com/v1alpha1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

const runnerSetKind = "autoscalingrunnerset"

var runnerSetGVK = schema.GroupVersionKind{
	Group:   "actions.github.com",
	Version: "v1alpha1",
	Kind:    "AutoscalingRunnerSet",
}

func (h *GithubAutoscalingRunnersets) init(client dynamic.Interface) {
	h.Client = client.Resource(schema.GroupVersionResource{
		Group:    "actions.github.com",
		Version:  "v1alpha1",
		Resource: "autoscalingrunnersets",
	})
}

// SetState sets the state of Github Autoscaling Runnersets resources based on the current period.
func (h *GithubAutoscalingRunnersets) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	scalerStatusSuccess := []common.ScalerStatusSuccess{}
	scalerStatusFailed := []common.ScalerStatusFailed{}
	list := []unstructured.Unstructured{}

	var err error

	for _, ns := range h.Resource.NsList {
		h.Logger.Debug().Msgf("found namespace: %s", ns)

		deployList, err := h.Client.Namespace(ns).List(ctx, h.Resource.ListOptions)
		if err != nil {
			h.Logger.Debug().Err(err).Msg("error listing autoscaling runner sets")

			return scalerStatusSuccess, scalerStatusFailed, fmt.Errorf("error listing autoscaling runner sets: %w", err)
		}

		list = append(list, deployList.Items...)
	}

	h.Logger.Debug().Msgf("number of autoscaling runner sets: %d", len(list))

	for _, item := range list {
		runnerSet := &actionsV1alpha1.AutoscalingRunnerSet{}
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, runnerSet); err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   item.GetKind(),
					Name:   item.GetName(),
					Reason: err.Error(),
				},
			)

			continue
		}

		switch h.Resource.Period.Type {
		case "down":
			h.Logger.Debug().Msgf("scaling down: %s", runnerSet.Name)

			runnerSet.Annotations = utils.AddMinMaxAnnotations(
				runnerSet.Annotations,
				h.Resource.Period,
				intPtrToInt32Ptr(runnerSet.Spec.MinRunners),
				int32(ptr.Deref(runnerSet.Spec.MaxRunners, 0)),
			)
			runnerSet.Spec.MinRunners = ptr.To(int(h.Resource.Period.MinReplicas))
			runnerSet.Spec.MaxRunners = ptr.To(int(h.Resource.Period.MaxReplicas))

		case "up":
			h.Logger.Debug().Msgf("scaling up: %s", runnerSet.Name)

			runnerSet.Annotations = utils.AddMinMaxAnnotations(
				runnerSet.Annotations,
				h.Resource.Period,
				intPtrToInt32Ptr(runnerSet.Spec.MinRunners),
				int32(ptr.Deref(runnerSet.Spec.MaxRunners, 0)),
			)
			runnerSet.Spec.MinRunners = ptr.To(int(h.Resource.Period.MinReplicas))
			runnerSet.Spec.MaxRunners = ptr.To(int(h.Resource.Period.MaxReplicas))

		default:
			h.Logger.Debug().Msgf("restoring: %s", runnerSet.Name)

			var (
				isAlreadyRestored bool
				minReplicas       *int32
				maxReplicas       int32
				annotations       map[string]string
			)

			isAlreadyRestored, minReplicas, maxReplicas, annotations, err =
				utils.RestoreMinMaxAnnotations(runnerSet.Annotations)
			if err != nil {
				scalerStatusFailed = append(
					scalerStatusFailed,
					common.ScalerStatusFailed{
						Kind:   runnerSetKind,
						Name:   runnerSet.Name,
						Reason: err.Error(),
					},
				)

				continue
			}

			if isAlreadyRestored {
				h.Logger.Debug().Msgf("nothing to do: %s", runnerSet.Name)
				continue
			}

			runnerSet.Annotations = annotations
			runnerSet.Spec.MinRunners = int32PtrToIntPtr(minReplicas)
			runnerSet.Spec.MaxRunners = int32ToIntPtr(maxReplicas)
		}

		h.Logger.Debug().Msgf("update autoscaling runner set: %s", runnerSet.Name)

		runnerSet.SetGroupVersionKind(runnerSetGVK)

		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(runnerSet)
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   runnerSetKind,
					Name:   runnerSet.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		_, err = h.Client.Namespace(runnerSet.Namespace).Update(
			ctx,
			&unstructured.Unstructured{Object: unstructuredObj},
			metaV1.UpdateOptions{
				FieldManager: utils.FieldManager,
			},
		)
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   runnerSetKind,
					Name:   runnerSet.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		scalerStatusSuccess = append(
			scalerStatusSuccess,
			common.ScalerStatusSuccess{
				Kind: runnerSetKind,
				Name: runnerSet.Name,
			},
		)
	}

	return scalerStatusSuccess, scalerStatusFailed, nil
}

func intPtrToInt32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}

	v := int32(*value)

	return &v
}

func int32PtrToIntPtr(value *int32) *int {
	if value == nil {
		return nil
	}

	v := int(*value)

	return &v
}

func int32ToIntPtr(value int32) *int {
	v := int(value)

	return &v
}
