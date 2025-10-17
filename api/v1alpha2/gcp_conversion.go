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

package v1alpha2

import (
	"log"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	kubecloudscalercloudv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

// ConvertTo converts this Gcp (v1alpha2) to the Hub version (v1alpha3).
func (src *Gcp) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*kubecloudscalercloudv1alpha3.Gcp)
	log.Printf("ConvertTo: Converting Gcp from Spoke version v1alpha2 to Hub version v1alpha3;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v1alpha2 to v1alpha3
	// Example: Copying Spec fields
	// dst.Spec.Size = src.Spec.Replicas

	// Copy ObjectMeta to preserve name, namespace, labels, etc.
	dst.ObjectMeta = src.ObjectMeta

	return nil
}

// ConvertFrom converts the Hub version (v1alpha3) to this Gcp (v1alpha2).
func (dst *Gcp) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*kubecloudscalercloudv1alpha3.Gcp)
	log.Printf("ConvertFrom: Converting Gcp from Hub version v1alpha3 to Spoke version v1alpha2;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v1alpha3 to v1alpha2
	// Example: Copying Spec fields
	// dst.Spec.Replicas = src.Spec.Size

	// Copy ObjectMeta to preserve name, namespace, labels, etc.
	dst.ObjectMeta = src.ObjectMeta

	return nil
}
