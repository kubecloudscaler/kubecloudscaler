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
	"log"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	kubecloudscalercloudv1alpha2 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha2"
)

// ConvertTo converts this Gcp (v1alpha1) to the Hub version (v1alpha2).
func (src *Gcp) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*kubecloudscalercloudv1alpha2.Gcp)
	log.Printf("ConvertTo: Converting Gcp from Spoke version v1alpha1 to Hub version v1alpha2;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.DryRun = src.Spec.DryRun
	dst.Spec.Periods = src.Spec.Periods
	dst.Spec.ProjectId = src.Spec.ProjectId
	dst.Spec.Region = src.Spec.Region
	dst.Spec.AuthSecret = src.Spec.AuthSecret
	dst.Spec.RestoreOnDelete = src.Spec.RestoreOnDelete
	dst.Spec.WaitForOperation = src.Spec.WaitForOperation
	dst.Spec.DefaultPeriodType = "down"

	// convert fields from v1alpha1 to v1alpha2
	dst.Spec.Resources.Types = src.Spec.Resources
	dst.Spec.Resources.LabelSelector = src.Spec.LabelSelector

	// Status
	dst.Status = src.Status

	return nil
}

// ConvertFrom converts the Hub version (v1alpha2) to this Gcp (v1alpha1).
func (dst *Gcp) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*kubecloudscalercloudv1alpha2.Gcp)
	log.Printf("ConvertFrom: Converting Gcp from Hub version v1alpha2 to Spoke version v1alpha1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.DryRun = src.Spec.DryRun
	dst.Spec.Periods = src.Spec.Periods
	dst.Spec.ProjectId = src.Spec.ProjectId
	dst.Spec.Region = src.Spec.Region
	dst.Spec.AuthSecret = src.Spec.AuthSecret
	dst.Spec.RestoreOnDelete = src.Spec.RestoreOnDelete
	dst.Spec.WaitForOperation = src.Spec.WaitForOperation

	// convert fields from v1alpha2 to v1alpha1
	dst.Spec.Resources = src.Spec.Resources.Types
	dst.Spec.LabelSelector = src.Spec.Resources.LabelSelector

	// Status
	dst.Status = src.Status

	return nil
}
