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
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	kubecloudscalercloudv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

// ConvertTo converts this K8s (v1alpha1) to the Hub version (v1alpha2).
func (src *K8s) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*kubecloudscalercloudv1alpha3.K8s)
	log.Debug().Msgf("ConvertTo: Converting K8s from Spoke version v1alpha1 to Hub version v1alpha3;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.DryRun = src.Spec.DryRun
	dst.Spec.Periods = src.Spec.Periods
	dst.Spec.Config.Namespaces = src.Spec.Namespaces
	dst.Spec.Config.ExcludeNamespaces = src.Spec.ExcludeNamespaces
	dst.Spec.Config.ForceExcludeSystemNamespaces = src.Spec.ForceExcludeSystemNamespaces
	dst.Spec.Config.DeploymentTimeAnnotation = src.Spec.DeploymentTimeAnnotation
	dst.Spec.Config.DisableEvents = src.Spec.DisableEvents
	dst.Spec.Config.AuthSecret = src.Spec.AuthSecret
	dst.Spec.Config.RestoreOnDelete = src.Spec.RestoreOnDelete

	// convert fields from v1alpha1 to v1alpha3
	dst.Spec.Resources = src.Spec.Resources

	// Status
	dst.Status = src.Status

	return nil
}

// ConvertFrom converts the Hub version (v1alpha2) to this K8s (v1alpha1).
func (dst *K8s) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*kubecloudscalercloudv1alpha3.K8s)
	log.Debug().Msgf("ConvertFrom: Converting K8s from Hub version v1alpha3 to Spoke version v1alpha1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec
	dst.Spec.DryRun = src.Spec.DryRun
	dst.Spec.Periods = src.Spec.Periods
	dst.Spec.Namespaces = src.Spec.Config.Namespaces
	dst.Spec.ExcludeNamespaces = src.Spec.Config.ExcludeNamespaces
	dst.Spec.ForceExcludeSystemNamespaces = src.Spec.Config.ForceExcludeSystemNamespaces
	dst.Spec.DeploymentTimeAnnotation = src.Spec.Config.DeploymentTimeAnnotation
	dst.Spec.DisableEvents = src.Spec.Config.DisableEvents
	dst.Spec.AuthSecret = src.Spec.Config.AuthSecret
	dst.Spec.RestoreOnDelete = src.Spec.Config.RestoreOnDelete

	// convert fields from v1alpha3 to v1alpha1
	dst.Spec.Resources = src.Spec.Resources

	// Status
	dst.Status = src.Status

	return nil
}
