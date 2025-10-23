package statefulsets

import (
	"context"
	"fmt"

	appsV1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
)

func (s *Statefulsets) init(client kubernetes.Interface) {
	s.Client = client.AppsV1()
}

// SetState sets the state of StatefulSet resources based on the current period.
func (s *Statefulsets) SetState(ctx context.Context) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error) {
	scalerStatusSuccess := []common.ScalerStatusSuccess{}
	scalerStatusFailed := []common.ScalerStatusFailed{}
	list := []appsV1.StatefulSet{}

	for _, ns := range s.Resource.NsList {
		s.Logger.Debug().Msgf("found namespace: %s", ns)

		statefulList, err := s.Client.StatefulSets(ns).List(ctx, s.Resource.ListOptions)
		if err != nil {
			s.Logger.Debug().Err(err).Msg("error listing statefulsets")

			return scalerStatusSuccess, scalerStatusFailed, fmt.Errorf("error listing statefulsets: %w", err)
		}

		list = append(list, statefulList.Items...)
	}

	s.Logger.Debug().Msgf("number of statefulsets: %d", len(list))

	//nolint:gocritic // Range iteration of struct is acceptable, using index would reduce readability
	for _, dName := range list {
		s.Logger.Debug().Msgf("resource-name: %s", dName.Name)
		var stateful *appsV1.StatefulSet

		stateful, err := s.Client.StatefulSets(dName.Namespace).Get(ctx, dName.Name, metaV1.GetOptions{})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "statefulset",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		if _, exists := stateful.Annotations[utils.AnnotationsPrefix+"/"+utils.AnnotationIgnore]; exists {
			scalerStatusSuccess = append(
				scalerStatusSuccess,
				common.ScalerStatusSuccess{
					Kind:    "statefulset",
					Name:    dName.Name,
					Comment: "ignored due to annotation",
				},
			)

			continue
		}

		switch s.Resource.Period.Type {
		case "down":
			s.Logger.Debug().Msgf("scaling down: %s", dName.Name)

			stateful.Annotations = utils.AddIntAnnotations(stateful.Annotations, s.Resource.Period, stateful.Spec.Replicas)

			stateful.Spec.Replicas = ptr.To(s.Resource.Period.MinReplicas)

		case "up":
			s.Logger.Debug().Msgf("scaling up: %s", dName.Name)

			stateful.Annotations = utils.AddIntAnnotations(stateful.Annotations, s.Resource.Period, stateful.Spec.Replicas)
			stateful.Spec.Replicas = ptr.To(s.Resource.Period.MaxReplicas)

		default:
			s.Logger.Debug().Msgf("restoring: %s", dName.Name)

			var isAlreadyRestored bool

			isAlreadyRestored, stateful.Spec.Replicas, stateful.Annotations, err = utils.RestoreIntAnnotations(stateful.Annotations)
			if err != nil {
				scalerStatusFailed = append(
					scalerStatusFailed,
					common.ScalerStatusFailed{
						Kind:   "statefulset",
						Name:   dName.Name,
						Reason: err.Error(),
					},
				)

				continue
			}

			if isAlreadyRestored {
				s.Logger.Debug().Msgf("nothing to do: %s", dName.Name)
				continue
			}
		}

		s.Logger.Debug().Msgf("update statefulset: %s", dName.Name)

		_, err = s.Client.StatefulSets(dName.Namespace).Update(ctx, stateful, metaV1.UpdateOptions{
			FieldManager: utils.FieldManager,
		})
		if err != nil {
			scalerStatusFailed = append(
				scalerStatusFailed,
				common.ScalerStatusFailed{
					Kind:   "statefulset",
					Name:   dName.Name,
					Reason: err.Error(),
				},
			)

			continue
		}

		scalerStatusSuccess = append(
			scalerStatusSuccess,
			common.ScalerStatusSuccess{
				Kind: "statefulset",
				Name: dName.Name,
			},
		)
	}

	return scalerStatusSuccess, scalerStatusFailed, nil
}
