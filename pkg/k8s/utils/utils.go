package utils

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	periodPkg "github.com/k8scloudscaler/k8scloudscaler/pkg/period"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// We set namespaces into resources to be able to use k8s client
func SetNamespaceList(ctx context.Context, config *Config) ([]string, error) {
	_ = log.FromContext(ctx)
	nsList := []string{}

	// get the list of namespaces
	if len(config.Namespaces) > 0 {
		nsList = config.Namespaces
	} else {
		// get all namespaces from the cluster
		nsListItems, err := config.Client.CoreV1().Namespaces().List(context.Background(), metaV1.ListOptions{})
		if err != nil {
			log.Log.V(1).Info("error listing namespaces")

			return []string{}, err
		}

		for _, ns := range nsListItems.Items {
			nsList = append(nsList, ns.Name)
		}
	}

	// exclude namespaces
	if len(config.ExcludeNamespaces) == 0 {
		config.ExcludeNamespaces = DefaultExcludeNamespaces
	}

	for _, ns := range config.ExcludeNamespaces {
		for i, n := range nsList {
			if slices.Contains(DefaultExcludeNamespaces, n) {
				continue
			}

			if n == ns {
				nsList = append(nsList[:i], nsList[i+1:]...)
			}
		}
	}

	if config.ForceExcludeSystemNamespaces {
		for _, ns := range DefaultExcludeNamespaces {
			for i, n := range nsList {
				if n == ns {
					nsList = append(nsList[:i], nsList[i+1:]...)
				}
			}
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

func RemoveAnnotations(annotations map[string]string) map[string]string {
	for annot := range annotations {
		if strings.HasPrefix(annot, AnnotationsPrefix) {
			delete(annotations, annot)
		}
	}

	return annotations
}

func PrepareSearch(ctx context.Context, config *Config) ([]string, metaV1.ListOptions, error) {
	_ = log.FromContext(ctx)
	var err error

	nsList, err := SetNamespaceList(ctx, config)
	if err != nil {
		log.Log.Error(err, "unable to set namespace list")

		return []string{}, metaV1.ListOptions{}, err
	}

	// set a default label selector to ignore resources with the label "k8scloudscaler/ignore"
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
		log.Log.V(1).Info("labelSelector", "selectors", config.LabelSelector)

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

func InitConfig(ctx context.Context, config *Config) (*K8sResource, error) {
	_ = log.FromContext(ctx)
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

func AddMinMaxAnnotations(annot map[string]string, curPeriod *periodPkg.Period, min *int32, max int32) map[string]string {
	annotations := addAnnotations(annot, curPeriod)

	_, isExists := annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsMinOrigValue] = strconv.FormatInt(int64(ptr.Deref(min, int32(0))), 10)
		annotations[AnnotationsPrefix+"/"+AnnotationsMaxOrigValue] = fmt.Sprintf("%d", max)
	}

	return annotations
}

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
			return true, nil, 0, annot, err
		}
	} else {
		isMinRestored = true
	}

	rep, isExists = annot[AnnotationsPrefix+"/"+AnnotationsMaxOrigValue]
	if isExists {
		maxAsInt, err = strconv.Atoi(rep)
		if err != nil {
			return true, nil, 0, annot, err
		}
	} else {
		isMaxRestored = true
	}

	isRestored := isMinRestored && isMaxRestored
	annot = RemoveAnnotations(annot)

	return isRestored, ptr.To(int32(minAsInt)), int32(maxAsInt), annot, nil
}

func AddBoolAnnotations(annot map[string]string, curPeriod *periodPkg.Period, value bool) map[string]string {
	annotations := addAnnotations(annot, curPeriod)

	_, isExists := annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue] = strconv.FormatBool(value)
	}

	return annotations
}

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
			return false, nil, annot, err
		}
	} else {
		isRestored = true
	}

	annot = RemoveAnnotations(annot)

	return isRestored, ptr.To(repAsBool), annot, nil
}

func AddIntAnnotations(annot map[string]string, curPeriod *periodPkg.Period, value *int32) map[string]string {
	annotations := addAnnotations(annot, curPeriod)

	_, isExists := annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue] = strconv.FormatInt(int64(ptr.Deref(value, int32(0))), 10)
	}

	return annotations
}

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
			return true, nil, annot, err
		}
	} else {
		isRestored = true
	}

	annot = RemoveAnnotations(annot)

	return isRestored, ptr.To(int32(repAsInt)), annot, nil
}
