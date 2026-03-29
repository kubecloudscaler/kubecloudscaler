// +kubebuilder:object:generate=false
// +kubebuilder:skip
package scaledobjects

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is the GVR for KEDA ScaledObjects.
var SchemeGroupVersion = schema.GroupVersion{Group: "keda.sh", Version: "v1alpha1"}

// AddToScheme registers ScaledObject types with the given scheme.
func AddToScheme(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&ScaledObject{},
		&ScaledObjectList{},
	)
	metaV1.AddToGroupVersion(s, SchemeGroupVersion)
	return nil
}

// ScaledObjectList is a list of ScaledObject resources.
type ScaledObjectList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`
	Items           []ScaledObject `json:"items"`
}

// DeepCopyObject implements runtime.Object.
func (l *ScaledObjectList) DeepCopyObject() runtime.Object {
	if l == nil {
		return nil
	}
	out := *l
	if l.Items != nil {
		out.Items = make([]ScaledObject, len(l.Items))
		for i := range l.Items {
			obj := l.Items[i].DeepCopyObject()
			out.Items[i] = *obj.(*ScaledObject) //nolint:errcheck // type assertion is guaranteed safe
		}
	}
	return &out
}

// ScaledObject mirrors the KEDA ScaledObject CRD (keda.sh/v1alpha1).
// Defined locally to avoid importing the full KEDA module which has
// build incompatibilities with the project's pinned K8s versions.
type ScaledObject struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ScaledObjectSpec `json:"spec,omitempty"`
}

// DeepCopyObject implements runtime.Object.
func (s *ScaledObject) DeepCopyObject() runtime.Object {
	if s == nil {
		return nil
	}
	out := *s
	s.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = *s.Spec.DeepCopy()
	return &out
}

// ScaledObjectSpec contains the spec fields of a KEDA ScaledObject.
// Only the fields needed for scaling are defined here.
type ScaledObjectSpec struct {
	MinReplicaCount *int32 `json:"minReplicaCount,omitempty"`
	MaxReplicaCount *int32 `json:"maxReplicaCount,omitempty"`
}

// DeepCopy returns a deep copy of ScaledObjectSpec.
func (s *ScaledObjectSpec) DeepCopy() *ScaledObjectSpec {
	if s == nil {
		return nil
	}
	out := *s
	if s.MinReplicaCount != nil {
		v := *s.MinReplicaCount
		out.MinReplicaCount = &v
	}
	if s.MaxReplicaCount != nil {
		v := *s.MaxReplicaCount
		out.MaxReplicaCount = &v
	}
	return &out
}
