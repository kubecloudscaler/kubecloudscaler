package utils

import (
	"context"
	"strings"

	"github.com/golgoth31/cloudscaler/api/common"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func AddAnnotations(annotations map[string]string, period *common.ScalerPeriod) map[string]string {
	annotations[AnnotationsPrefix+"/"+PeriodType] = period.Type
	annotations[AnnotationsPrefix+"/"+PeriodStartTime] = period.Time.StartTime
	annotations[AnnotationsPrefix+"/"+PeriodEndTime] = period.Time.EndTime
	annotations[AnnotationsPrefix+"/"+PeriodTimezone] = period.Time.Timezone

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
