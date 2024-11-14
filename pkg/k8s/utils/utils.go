package utils

import (
	"context"
	"strconv"
	"strings"

	periodPkg "github.com/cloudscalerio/cloudscaler/pkg/period"
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
		config.ExcludeNamespaces = DefaultExcludeNamespace
	}

	for _, ns := range config.ExcludeNamespaces {
		for i, n := range nsList {
			if n == ns {
				nsList = append(nsList[:i], nsList[i+1:]...)
			}
		}
	}

	return nsList, nil
}

func addAnnotations(annotations map[string]string, period *periodPkg.Period, value string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[AnnotationsPrefix+"/"+PeriodType] = period.Type
	annotations[AnnotationsPrefix+"/"+PeriodStartTime] = period.GetStartTime.String()
	annotations[AnnotationsPrefix+"/"+PeriodEndTime] = period.GetEndTime.String()
	ptr.Deref(period.Period.Timezone, annotations[AnnotationsPrefix+"/"+PeriodTimezone])

	_, isExists := annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue] = value
	}

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

	listOptions := metaV1.ListOptions{}
	if config.LabelSelector != nil {
		log.Log.V(1).Info("labelSelector", "selectors", config.LabelSelector)

		listOptions = metaV1.ListOptions{
			LabelSelector: metaV1.FormatLabelSelector(config.LabelSelector),
		}
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

func AddBoolAnnotations(annot map[string]string, curPeriod *periodPkg.Period, value bool) map[string]string {
	return addAnnotations(annot, curPeriod, strconv.FormatBool(value))
}

func RestoreBool(annot map[string]string) (*bool, map[string]string, error) {
	var (
		repAsBool bool
		err       error
	)

	rep, isExists := annot[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if isExists {
		repAsBool, err = strconv.ParseBool(rep)
		if err != nil {
			return nil, annot, err
		}
	}

	annot = RemoveAnnotations(annot)

	return ptr.To(repAsBool), annot, nil
}

func AddIntAnnotations(annot map[string]string, curPeriod *periodPkg.Period, value *int32) map[string]string {
	return addAnnotations(annot, curPeriod, strconv.FormatInt(int64(ptr.Deref(value, int32(0))), 10))
}

func RestoreInt(annot map[string]string) (*int32, map[string]string, error) {
	var (
		repAsInt int
		err      error
	)

	rep, isExists := annot[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if isExists {
		repAsInt, err = strconv.Atoi(rep)
		if err != nil {
			return nil, annot, err
		}
	}

	annot = RemoveAnnotations(annot)

	return ptr.To(int32(repAsInt)), annot, nil
}
