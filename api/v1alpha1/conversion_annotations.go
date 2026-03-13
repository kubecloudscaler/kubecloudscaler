/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"encoding/json"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
)

// Annotation keys for preserving v1alpha1-specific fields during conversion round-trips.
// Fields that exist in v1alpha1 but not in v1alpha3 (or vice versa) are stored as
// annotations on the hub (v1alpha3) object so they survive the round trip.
const (
	annotationPrefix                      = "kubecloudscaler.cloud/conversion-v1alpha1-"
	annotationExcludeResources            = annotationPrefix + "excludeResources"
	annotationGCPDeploymentTimeAnnotation = annotationPrefix + "gcp-deploymentTimeAnnotation"
	annotationGCPDefaultPeriodType        = annotationPrefix + "gcp-defaultPeriodType"
)

// setConversionAnnotation sets a conversion annotation on the ObjectMeta.
func setConversionAnnotation(meta *metav1.ObjectMeta, key, value string) {
	if value == "" {
		return
	}
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	meta.Annotations[key] = value
}

// getConversionAnnotation reads and removes a conversion annotation from the ObjectMeta.
func getConversionAnnotation(meta *metav1.ObjectMeta, key string) string {
	if meta.Annotations == nil {
		return ""
	}
	v := meta.Annotations[key]
	delete(meta.Annotations, key)
	if len(meta.Annotations) == 0 {
		meta.Annotations = nil
	}
	return v
}

// encodeStringSlice encodes a string slice as a comma-separated value for annotation storage.
func encodeStringSlice(s []string) string {
	if len(s) == 0 {
		return ""
	}
	data, err := json.Marshal(s)
	if err != nil {
		return strings.Join(s, ",")
	}
	return string(data)
}

// decodeStringSlice decodes a string slice from an annotation value.
func decodeStringSlice(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return strings.Split(s, ",")
	}
	return result
}

// stringsToResourceKinds converts []string to []common.ResourceKind for v1alpha1→v1alpha3 conversion.
func stringsToResourceKinds(ss []string) []common.ResourceKind {
	if ss == nil {
		return nil
	}
	kinds := make([]common.ResourceKind, len(ss))
	for i, s := range ss {
		kinds[i] = common.ResourceKind(s)
	}
	return kinds
}

// resourceKindsToStrings converts []common.ResourceKind to []string for v1alpha3→v1alpha1 conversion.
func resourceKindsToStrings(kinds []common.ResourceKind) []string {
	if kinds == nil {
		return nil
	}
	ss := make([]string, len(kinds))
	for i, k := range kinds {
		ss[i] = string(k)
	}
	return ss
}
