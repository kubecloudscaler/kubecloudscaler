// Package utils provides utility functions for Kubernetes resource management.
package utils

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// SetNamespaceList sets namespaces into resources to be able to use k8s client.
func SetNamespaceList(ctx context.Context, config *Config) ([]string, error) {
	logger := zerolog.Ctx(ctx)
	nsList := []string{}

	// get the list of namespaces
	if len(config.Namespaces) > 0 {
		nsList = config.Namespaces
	} else {
		// get all namespaces from the cluster
		nsListItems, err := config.Client.CoreV1().Namespaces().List(ctx, metaV1.ListOptions{})
		if err != nil {
			logger.Debug().Msg("error listing namespaces")

			return []string{}, fmt.Errorf("error listing namespaces: %w", err)
		}

		for _, ns := range nsListItems.Items {
			if slices.Contains(config.ExcludeNamespaces, ns.Name) {
				continue
			}

			nsList = append(nsList, ns.Name)
		}
	}

	// force exclude system namespaces
	if config.ForceExcludeSystemNamespaces {
		for _, ns := range DefaultExcludeNamespaces {
			for i, n := range nsList {
				if n == ns {
					nsList = append(nsList[:i], nsList[i+1:]...)
				}
			}
		}
	}

	// force always exclude my own namespace
	for i, n := range nsList {
		if n == os.Getenv("POD_NAMESPACE") {
			nsList = append(nsList[:i], nsList[i+1:]...)
		}
	}

	return nsList, nil
}

func addAnnotations(annotations map[string]string, period *periodPkg.Period) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[AnnotationsPrefix+"/"+PeriodType] = period.Type
	annotations[AnnotationsPrefix+"/"+PeriodStartTime] = period.GetStartTime.String()
	annotations[AnnotationsPrefix+"/"+PeriodEndTime] = period.GetEndTime.String()
	ptr.Deref(period.Period.Timezone, annotations[AnnotationsPrefix+"/"+PeriodTimezone])

	return annotations
}

// RemoveAnnotations removes all kubecloudscaler annotations from the given map.
func RemoveAnnotations(annotations map[string]string) map[string]string {
	for annot := range annotations {
		if strings.HasPrefix(annot, AnnotationsPrefix) {
			delete(annotations, annot)
		}
	}

	return annotations
}

// PrepareSearch prepares search parameters for Kubernetes resource queries.
func PrepareSearch(ctx context.Context, config *Config) ([]string, metaV1.ListOptions, error) {
	logger := zerolog.Ctx(ctx)
	var err error

	nsList, err := SetNamespaceList(ctx, config)
	if err != nil {
		logger.Error().Err(err).Msg("unable to set namespace list")

		return []string{}, metaV1.ListOptions{}, err
	}

	// set a default label selector to ignore resources with the label "kubecloudscaler/ignore"
	labelSelectors := metaV1.LabelSelector{
		MatchLabels: make(map[string]string),
		MatchExpressions: []metaV1.LabelSelectorRequirement{
			{
				Key:      AnnotationsPrefix + "/ignore",
				Operator: metaV1.LabelSelectorOpDoesNotExist,
			},
		},
	}

	if config.LabelSelector != nil {
		logger.Debug().Msgf("labelSelector: %+v", config.LabelSelector)

		if config.LabelSelector.MatchLabels != nil {
			for k, v := range config.LabelSelector.MatchLabels {
				labelSelectors.MatchLabels[k] = v
			}
		}

		if config.LabelSelector.MatchExpressions != nil {
			for _, v := range config.LabelSelector.MatchExpressions {
				if v.Key == AnnotationsPrefix+"/ignore" {
					continue
				}

				labelSelectors.MatchExpressions = append(labelSelectors.MatchExpressions, v)
			}
		}
	}

	listOptions := metaV1.ListOptions{
		LabelSelector: metaV1.FormatLabelSelector(&labelSelectors),
	}

	return nsList, listOptions, nil
}

// InitConfig initializes a K8sResource from the given configuration.
func InitConfig(ctx context.Context, config *Config) (*K8sResource, error) {
	resource := &K8sResource{
		Period: config.Period,
	}

	nsList, listOptions, err := PrepareSearch(ctx, config)
	if err != nil {
		return nil, err
	}

	resource.NsList = nsList
	resource.ListOptions = listOptions

	return resource, nil
}

// AddMinMaxAnnotations adds min/max replica annotations to the given map.
func AddMinMaxAnnotations(annot map[string]string, curPeriod *periodPkg.Period, minReplicas *int32, max int32) map[string]string {
	annotations := addAnnotations(annot, curPeriod)

	_, isExists := annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsMinOrigValue] = strconv.FormatInt(int64(ptr.Deref(minReplicas, int32(0))), 10)
		annotations[AnnotationsPrefix+"/"+AnnotationsMaxOrigValue] = fmt.Sprintf("%d", max)
	}

	return annotations
}

// RestoreMinMaxAnnotations restores min/max replica annotations from the given map.
func RestoreMinMaxAnnotations(annot map[string]string) (bool, *int32, int32, map[string]string, error) {
	var (
		minAsInt      int
		maxAsInt      int
		err           error
		isMinRestored bool
		isMaxRestored bool
	)

	rep, isExists := annot[AnnotationsPrefix+"/"+AnnotationsMinOrigValue]
	if isExists {
		minAsInt, err = strconv.Atoi(rep)
		if err != nil {
			return true, nil, 0, annot, fmt.Errorf("error parsing min value: %w", err)
		}
	} else {
		isMinRestored = true
	}

	rep, isExists = annot[AnnotationsPrefix+"/"+AnnotationsMaxOrigValue]
	if isExists {
		maxAsInt, err = strconv.Atoi(rep)
		if err != nil {
			return true, nil, 0, annot, fmt.Errorf("error parsing max value: %w", err)
		}
	} else {
		isMaxRestored = true
	}

	isRestored := isMinRestored && isMaxRestored
	annot = RemoveAnnotations(annot)

	//nolint:gosec // G109: int32 conversion is safe for replica count values which are bounded
	return isRestored, ptr.To(int32(minAsInt)), int32(maxAsInt), annot, nil
}

// AddBoolAnnotations adds boolean value annotations to the given map.
func AddBoolAnnotations(annot map[string]string, curPeriod *periodPkg.Period, value bool) map[string]string {
	annotations := addAnnotations(annot, curPeriod)

	_, isExists := annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue] = strconv.FormatBool(value)
	}

	return annotations
}

// RestoreBoolAnnotations restores boolean value annotations from the given map.
func RestoreBoolAnnotations(annot map[string]string) (bool, *bool, map[string]string, error) {
	var (
		repAsBool  bool
		err        error
		isRestored bool
	)

	rep, isExists := annot[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if isExists {
		repAsBool, err = strconv.ParseBool(rep)
		if err != nil {
			return false, nil, annot, fmt.Errorf("error parsing bool value: %w", err)
		}
	} else {
		isRestored = true
	}

	annot = RemoveAnnotations(annot)

	return isRestored, ptr.To(repAsBool), annot, nil
}

// AddIntAnnotations adds integer value annotations to the given map.
func AddIntAnnotations(annot map[string]string, curPeriod *periodPkg.Period, value *int32) map[string]string {
	annotations := addAnnotations(annot, curPeriod)

	_, isExists := annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue] = strconv.FormatInt(int64(ptr.Deref(value, int32(0))), 10)
	}

	return annotations
}

// RestoreIntAnnotations restores integer value annotations from the given map.
func RestoreIntAnnotations(annot map[string]string) (bool, *int32, map[string]string, error) {
	var (
		repAsInt   int
		err        error
		isRestored bool
	)

	rep, isExists := annot[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if isExists {
		repAsInt, err = strconv.Atoi(rep)
		if err != nil {
			return true, nil, annot, fmt.Errorf("error parsing int value: %w", err)
		}
	} else {
		isRestored = true
	}

	annot = RemoveAnnotations(annot)

	//nolint:gosec // G109: int32 conversion is safe for replica count values which are bounded
	return isRestored, ptr.To(int32(repAsInt)), annot, nil
}
