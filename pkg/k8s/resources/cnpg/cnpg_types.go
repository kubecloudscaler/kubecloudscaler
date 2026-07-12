// Package cnpg includes local CloudNativePG type shims for listers and adapters.
//
// +kubebuilder:object:generate=false
// +kubebuilder:skip
package cnpg

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is the GVR for CloudNativePG Clusters.
var SchemeGroupVersion = schema.GroupVersion{Group: "postgresql.cnpg.io", Version: "v1"}

// AddToScheme registers Cluster types with the given scheme.
func AddToScheme(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&Cluster{},
		&ClusterList{},
	)
	metaV1.AddToGroupVersion(s, SchemeGroupVersion)
	return nil
}

// ClusterList is a list of Cluster resources.
type ClusterList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

// DeepCopyObject implements runtime.Object.
func (l *ClusterList) DeepCopyObject() runtime.Object {
	if l == nil {
		return nil
	}
	out := *l
	if l.Items != nil {
		out.Items = make([]Cluster, len(l.Items))
		for i := range l.Items {
			obj := l.Items[i].DeepCopyObject()
			out.Items[i] = *obj.(*Cluster) //nolint:errcheck // type assertion is guaranteed safe
		}
	}
	return &out
}

// Cluster mirrors the CloudNativePG Cluster CRD (postgresql.cnpg.io/v1).
// Defined locally to avoid importing the full cloudnative-pg module. Hibernation
// is driven entirely by the cnpg.io/hibernation annotation on metadata, so only
// ObjectMeta is needed here; no spec fields are mirrored.
type Cluster struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`
}

// DeepCopyObject implements runtime.Object.
func (c *Cluster) DeepCopyObject() runtime.Object {
	if c == nil {
		return nil
	}
	out := *c
	c.DeepCopyInto(&out.ObjectMeta)
	return &out
}
