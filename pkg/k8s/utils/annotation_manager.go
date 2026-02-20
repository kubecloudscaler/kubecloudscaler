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

// Package utils provides annotation management functionality for Kubernetes resources.
package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"k8s.io/utils/ptr"

	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// annotationManager implements AnnotationManager interface
type annotationManager struct{}

// NewAnnotationManager creates a new annotation manager
func NewAnnotationManager() AnnotationManager {
	return &annotationManager{}
}

// AddAnnotations adds period annotations to the annotations map
func (am *annotationManager) AddAnnotations(annotations map[string]string, period interface{}) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	switch p := period.(type) {
	case *periodPkg.Period:
		annotations[AnnotationsPrefix+"/"+PeriodType] = p.Type
		annotations[AnnotationsPrefix+"/"+PeriodStartTime] = p.GetStartTime.Format(time.RFC3339)
		annotations[AnnotationsPrefix+"/"+PeriodEndTime] = p.GetEndTime.Format(time.RFC3339)
		if p.Period != nil && p.Period.Timezone != nil {
			annotations[AnnotationsPrefix+"/"+PeriodTimezone] = *p.Period.Timezone
		}
	case interface {
		GetType() string
		GetStartTime() interface{ String() string }
		GetEndTime() interface{ String() string }
		GetTimezone() *string
	}:
		annotations[AnnotationsPrefix+"/"+PeriodType] = p.GetType()
		annotations[AnnotationsPrefix+"/"+PeriodStartTime] = p.GetStartTime().String()
		annotations[AnnotationsPrefix+"/"+PeriodEndTime] = p.GetEndTime().String()
		if timezone := p.GetTimezone(); timezone != nil {
			annotations[AnnotationsPrefix+"/"+PeriodTimezone] = *timezone
		}
	}

	return annotations
}

// RemoveAnnotations removes all kubecloudscaler annotations from the map
func (am *annotationManager) RemoveAnnotations(annotations map[string]string) map[string]string {
	for annot := range annotations {
		if strings.HasPrefix(annot, AnnotationsPrefix) {
			delete(annotations, annot)
		}
	}
	return annotations
}

// AddMinMaxAnnotations adds annotations for minimum and maximum replicas.
//
//nolint:revive,gocritic // maxReplicas parameter name is clearer than renaming to avoid builtin 'max'
func (am *annotationManager) AddMinMaxAnnotations(
	annot map[string]string,
	curPeriod interface{},
	minReplicas *int32,
	max int32,
) map[string]string {
	annotations := am.AddAnnotations(annot, curPeriod)

	_, isExists := annotations[AnnotationsPrefix+"/"+AnnotationsMinOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsMinOrigValue] = strconv.FormatInt(int64(ptr.Deref(minReplicas, int32(0))), 10)
	}

	_, isExists = annotations[AnnotationsPrefix+"/"+AnnotationsMaxOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsMaxOrigValue] = fmt.Sprintf("%d", max)
	}

	return annotations
}

// RestoreMinMaxAnnotations restores min/max values from annotations
//
//nolint:gocritic // Multiple return values needed for clear API interface
func (am *annotationManager) RestoreMinMaxAnnotations(annot map[string]string) (bool, *int32, int32, map[string]string, error) {
	var (
		minAsInt      int
		maxAsInt      int
		err           error
		isMinRestored bool
		isMaxRestored bool
	)

	rep, isExists := annot[AnnotationsPrefix+"/"+AnnotationsMinOrigValue]
	if isExists {
		minAsInt, err = strconv.Atoi(rep)
		if err != nil {
			return true, nil, 0, annot, fmt.Errorf("error parsing min value: %w", err)
		}
	} else {
		isMinRestored = true
	}

	rep, isExists = annot[AnnotationsPrefix+"/"+AnnotationsMaxOrigValue]
	if isExists {
		maxAsInt, err = strconv.Atoi(rep)
		if err != nil {
			return true, nil, 0, annot, fmt.Errorf("error parsing max value: %w", err)
		}
	} else {
		isMaxRestored = true
	}

	isRestored := isMinRestored && isMaxRestored
	annot = am.RemoveAnnotations(annot)

	//nolint:gosec // G109: int32 conversion is safe for replica count values which are bounded
	return isRestored, ptr.To(int32(minAsInt)), int32(maxAsInt), annot, nil
}

// AddBoolAnnotations adds bool annotation with original value
func (am *annotationManager) AddBoolAnnotations(annot map[string]string, curPeriod interface{}, value bool) map[string]string {
	annotations := am.AddAnnotations(annot, curPeriod)

	_, isExists := annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue] = strconv.FormatBool(value)
	}

	return annotations
}

// RestoreBoolAnnotations restores bool value from annotations
//
//nolint:gocritic // Multiple return values needed for clear API interface
func (am *annotationManager) RestoreBoolAnnotations(annot map[string]string) (bool, *bool, map[string]string, error) {
	var (
		repAsBool  bool
		err        error
		isRestored bool
	)

	rep, isExists := annot[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if isExists {
		repAsBool, err = strconv.ParseBool(rep)
		if err != nil {
			return false, nil, annot, fmt.Errorf("error parsing bool value: %w", err)
		}
	} else {
		isRestored = true
	}

	annot = am.RemoveAnnotations(annot)

	return isRestored, ptr.To(repAsBool), annot, nil
}

// AddIntAnnotations adds int annotation with original value
func (am *annotationManager) AddIntAnnotations(annot map[string]string, curPeriod interface{}, value *int32) map[string]string {
	annotations := am.AddAnnotations(annot, curPeriod)

	_, isExists := annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if !isExists {
		annotations[AnnotationsPrefix+"/"+AnnotationsOrigValue] = strconv.FormatInt(int64(ptr.Deref(value, int32(0))), 10)
	}

	return annotations
}

// RestoreIntAnnotations restores int value from annotations
//
//nolint:gocritic // Multiple return values needed for clear API interface
func (am *annotationManager) RestoreIntAnnotations(annot map[string]string) (bool, *int32, map[string]string, error) {
	var (
		repAsInt   int
		err        error
		isRestored bool
	)

	rep, isExists := annot[AnnotationsPrefix+"/"+AnnotationsOrigValue]
	if isExists {
		repAsInt, err = strconv.Atoi(rep)
		if err != nil {
			// Return false: the annotation exists but is corrupted, restoration has not happened.
			return false, nil, annot, fmt.Errorf("error parsing int value: %w", err)
		}
	} else {
		isRestored = true
	}

	annot = am.RemoveAnnotations(annot)

	//nolint:gosec // G109: int32 conversion is safe for replica count values which are bounded
	return isRestored, ptr.To(int32(repAsInt)), annot, nil
}
