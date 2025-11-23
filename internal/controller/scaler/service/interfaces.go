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

package service

import (
	"context"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// PeriodValidator defines the interface for validating periods.
type PeriodValidator interface {
	ValidatePeriod(
		periods []*common.ScalerPeriod,
		status *common.ScalerStatus,
		forceRestore bool,
	) (*periodPkg.Period, error)
}

// ResourceProcessor defines the interface for processing resources.
type ResourceProcessor interface {
	ProcessResources(
		ctx context.Context,
		resourceList []string,
		resourceConfig interface{},
	) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error)
}

// ScalerProcessor defines the interface for processing scaler resources.
type ScalerProcessor interface {
	ProcessScaler(
		ctx context.Context,
		scaler interface{},
		resourceList []string,
		resourceConfig interface{},
	) ([]common.ScalerStatusSuccess, []common.ScalerStatusFailed, error)
}
