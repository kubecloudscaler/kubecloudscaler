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
	"github.com/kubecloudscaler/kubecloudscaler/api/common"
	"github.com/kubecloudscaler/kubecloudscaler/internal/utils"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// PeriodValidatorService handles period validation for scalers.
type PeriodValidatorService struct{}

// NewPeriodValidatorService creates a new PeriodValidatorService.
func NewPeriodValidatorService() *PeriodValidatorService {
	return &PeriodValidatorService{}
}

// ValidatePeriod validates and returns the active period.
func (p *PeriodValidatorService) ValidatePeriod(
	periods []*common.ScalerPeriod,
	status *common.ScalerStatus,
	forceRestore bool,
) (*periodPkg.Period, error) {
	return utils.ValidatePeriod(periods, status, forceRestore)
}
