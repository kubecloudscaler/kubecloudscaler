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

package v1alpha3

import (
	"fmt"

	"github.com/kubecloudscaler/kubecloudscaler/api/common"
)

// validatePeriod validates a single ScalerPeriod configuration, wrapping errors with the period index.
func validatePeriod(p common.ScalerPeriod, index int) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("periods[%d]: %w", index, err)
	}

	return nil
}
