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

package utils

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BenchmarkSetNamespaceList benchmarks the SetNamespaceList function
func BenchmarkSetNamespaceList(b *testing.B) {
	ctx := context.Background()
	logger := zerolog.Nop()

	// Create a fake client with many namespaces
	namespaces := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		namespaces[i] = "namespace-" + string(rune(i))
	}

	client := NewFakeKubernetesClient()
	namespaceMgr := NewNamespaceManager(client, logger)

	config := &Config{
		Namespaces:                   namespaces,
		ForceExcludeSystemNamespaces: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := namespaceMgr.SetNamespaceList(ctx, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAddAnnotations benchmarks the AddAnnotations function
func BenchmarkAddAnnotations(b *testing.B) {
	annotationMgr := NewAnnotationManager()

	period := &MockPeriod{
		Type:      "test-period",
		StartTime: &mockTime{timeStr: "2024-01-01T00:00:00Z"},
		EndTime:   &mockTime{timeStr: "2024-01-01T01:00:00Z"},
		Timezone:  nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		annotations := make(map[string]string)
		annotationMgr.AddAnnotations(annotations, period)
	}
}

// BenchmarkRemoveAnnotations benchmarks the RemoveAnnotations function
func BenchmarkRemoveAnnotations(b *testing.B) {
	annotationMgr := NewAnnotationManager()

	// Create annotations with many kubecloudscaler annotations
	annotations := make(map[string]string)
	for i := 0; i < 100; i++ {
		annotations[AnnotationsPrefix+"/key-"+string(rune(i))] = "value"
		annotations["other-key-"+string(rune(i))] = "value"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a copy for each iteration
		testAnnotations := make(map[string]string)
		for k, v := range annotations {
			testAnnotations[k] = v
		}
		annotationMgr.RemoveAnnotations(testAnnotations)
	}
}

// BenchmarkAddMinMaxAnnotations benchmarks the AddMinMaxAnnotations function
func BenchmarkAddMinMaxAnnotations(b *testing.B) {
	annotationMgr := NewAnnotationManager()

	period := &MockPeriod{
		Type:      "test-period",
		StartTime: &mockTime{timeStr: "2024-01-01T00:00:00Z"},
		EndTime:   &mockTime{timeStr: "2024-01-01T01:00:00Z"},
		Timezone:  nil,
	}

	min := int32(2)
	max := int32(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		annotations := make(map[string]string)
		annotationMgr.AddMinMaxAnnotations(annotations, period, &min, max)
	}
}

// BenchmarkRestoreMinMaxAnnotations benchmarks the RestoreMinMaxAnnotations function
func BenchmarkRestoreMinMaxAnnotations(b *testing.B) {
	annotationMgr := NewAnnotationManager()

	annotations := map[string]string{
		AnnotationsPrefix + "/" + AnnotationsMinOrigValue: "5",
		AnnotationsPrefix + "/" + AnnotationsMaxOrigValue: "20",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a copy for each iteration
		testAnnotations := make(map[string]string)
		for k, v := range annotations {
			testAnnotations[k] = v
		}
		_, _, _, _, err := annotationMgr.RestoreMinMaxAnnotations(testAnnotations)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPrepareSearch benchmarks the PrepareSearch function
func BenchmarkPrepareSearch(b *testing.B) {
	ctx := context.Background()
	logger := zerolog.Nop()

	client := NewFakeKubernetesClient()
	namespaceMgr := NewNamespaceManager(client, logger)

	config := &Config{
		Namespaces:                   []string{"test-ns-1", "test-ns-2", "test-ns-3"},
		ForceExcludeSystemNamespaces: true,
		LabelSelector: &metaV1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "test",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := namespaceMgr.PrepareSearch(ctx, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkInitConfig benchmarks the InitConfig function
func BenchmarkInitConfig(b *testing.B) {
	ctx := context.Background()
	logger := zerolog.Nop()

	client := NewFakeKubernetesClient()
	namespaceMgr := NewNamespaceManager(client, logger)

	config := &Config{
		Namespaces:                   []string{"test-ns-1", "test-ns-2", "test-ns-3"},
		ForceExcludeSystemNamespaces: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := namespaceMgr.InitConfig(ctx, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}
