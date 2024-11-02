package utils

import (
	periodPkg "github.com/golgoth31/cloudscaler/pkg/period"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sResource struct {
	Config      *Config
	NsList      []string
	ListOptions metaV1.ListOptions
}

type Config struct {
	Namespaces        []string              `json:"namespaces,omitempty"`
	ExcludeNamespaces []string              `json:"excludeNamespaces,omitempty"`
	Client            *kubernetes.Clientset `json:"client"`
	LabelSelector     *metaV1.LabelSelector `json:"labelSelector,omitempty"`
	Period            *periodPkg.Period     `json:"period,omitempty"`
}
