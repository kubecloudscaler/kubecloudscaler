package utils

import (
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sResource struct {
	NsList      []string
	ListOptions metaV1.ListOptions
	Period      *periodPkg.Period `json:"period,omitempty"`
}

type Config struct {
	Namespaces                   []string              `json:"namespaces,omitempty"`
	ExcludeNamespaces            []string              `json:"excludeNamespaces,omitempty"`
	Client                       kubernetes.Interface  `json:"client"`
	LabelSelector                *metaV1.LabelSelector `json:"labelSelector,omitempty"`
	Period                       *periodPkg.Period     `json:"period,omitempty"`
	ForceExcludeSystemNamespaces bool                  `json:"forceExcludeSystemNamespaces,omitempty"`
}
