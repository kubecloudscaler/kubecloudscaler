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
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	kubecloudscalercloudv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

// ConvertTo converts this Gcp (v1alpha1) to the Hub version (v1alpha2).
func (src *Gcp) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*kubecloudscalercloudv1alpha3.Gcp)
	log.Debug().Msgf("ConvertTo: Converting Gcp from Spoke version v1alpha1 to Hub version v1alpha3;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.DryRun = src.Spec.DryRun
	// Convert []*common.ScalerPeriod to []common.ScalerPeriod
	dst.Spec.Periods = make([]common.ScalerPeriod, len(src.Spec.Periods))
	for i, period := range src.Spec.Periods {
		dst.Spec.Periods[i] = *period
	}
	dst.Spec.Config.ProjectId = src.Spec.ProjectId
	dst.Spec.Config.Region = src.Spec.Region
	dst.Spec.Config.AuthSecret = src.Spec.AuthSecret
	dst.Spec.Config.RestoreOnDelete = src.Spec.RestoreOnDelete
	dst.Spec.Config.WaitForOperation = src.Spec.WaitForOperation
	dst.Spec.Config.DefaultPeriodType = "down"

	// convert fields from v1alpha1 to v1alpha2
	dst.Spec.Resources.Types = src.Spec.Resources
	dst.Spec.Resources.LabelSelector = src.Spec.LabelSelector

	// Status
	dst.Status = src.Status

	return nil
}

// ConvertFrom converts the Hub version (v1alpha2) to this Gcp (v1alpha1).
func (dst *Gcp) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*kubecloudscalercloudv1alpha3.Gcp)
	log.Debug().Msgf("ConvertFrom: Converting Gcp from Hub version v1alpha3 to Spoke version v1alpha1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.DryRun = src.Spec.DryRun
	// Convert []common.ScalerPeriod to []*common.ScalerPeriod
	dst.Spec.Periods = make([]*common.ScalerPeriod, len(src.Spec.Periods))
	for i := range src.Spec.Periods {
		dst.Spec.Periods[i] = &src.Spec.Periods[i]
	}
	dst.Spec.ProjectId = src.Spec.Config.ProjectId
	dst.Spec.Region = src.Spec.Config.Region
	dst.Spec.AuthSecret = src.Spec.Config.AuthSecret
	dst.Spec.RestoreOnDelete = src.Spec.Config.RestoreOnDelete
	dst.Spec.WaitForOperation = src.Spec.Config.WaitForOperation

	// convert fields from v1alpha2 to v1alpha1
	dst.Spec.Resources = src.Spec.Resources.Types
	dst.Spec.LabelSelector = src.Spec.Resources.LabelSelector

	// Status
	dst.Status = src.Status

	return nil
}
