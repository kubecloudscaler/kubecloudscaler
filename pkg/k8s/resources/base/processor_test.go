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

package base

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubecloudscaler/kubecloudscaler/pkg/k8s/utils"
	periodPkg "github.com/kubecloudscaler/kubecloudscaler/pkg/period"
)

// --- Mock implementations ---

type mockLister struct {
	listFn func(ctx context.Context, namespace string, opts metaV1.ListOptions) ([]ResourceItem, error)
}

func (m *mockLister) List(
	ctx context.Context, namespace string, opts metaV1.ListOptions,
) ([]ResourceItem, error) {
	return m.listFn(ctx, namespace, opts)
}

type mockGetter struct {
	getFn func(ctx context.Context, namespace, name string, opts metaV1.GetOptions) (ResourceItem, error)
}

func (m *mockGetter) Get(
	ctx context.Context, namespace, name string, opts metaV1.GetOptions,
) (ResourceItem, error) {
	return m.getFn(ctx, namespace, name, opts)
}

type mockUpdater struct {
	updateFn func(ctx context.Context, namespace string, resource ResourceItem, opts metaV1.UpdateOptions) (ResourceItem, error)
}

func (m *mockUpdater) Update(
	ctx context.Context, namespace string, resource ResourceItem, opts metaV1.UpdateOptions,
) (ResourceItem, error) {
	return m.updateFn(ctx, namespace, resource, opts)
}

type mockStrategy struct {
	applyScalingFn func(ctx context.Context, resource ResourceItem, periodType string, period *periodPkg.Period) (bool, error)
	kind           string
}

func (m *mockStrategy) ApplyScaling(
	ctx context.Context, resource ResourceItem, periodType string, period *periodPkg.Period,
) (bool, error) {
	return m.applyScalingFn(ctx, resource, periodType, period)
}

func (m *mockStrategy) GetKind() string { return m.kind }

// --- Helper ---

func newTestProcessor(
	lister ResourceLister,
	getter ResourceGetter,
	updater ResourceUpdater,
	strategy ScalingStrategy,
	resource *utils.K8sResource,
) *Processor {
	return NewProcessor(lister, getter, updater, strategy, resource, testLogger())
}

func newItem(name, namespace string) *mockResourceItem {
	return &mockResourceItem{name: name, namespace: namespace, annotations: map[string]string{}}
}

// --- Tests ---

func TestProcessResources(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	tests := []struct {
		name            string
		resource        *utils.K8sResource
		lister          *mockLister
		getter          *mockGetter
		updater         *mockUpdater
		strategy        *mockStrategy
		wantSuccessLen  int
		wantFailedLen   int
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "lister error returns error",
			resource: &utils.K8sResource{
				NsList: []string{"default"},
				Period: &periodPkg.Period{},
			},
			lister: &mockLister{
				listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
					return nil, errTest
				},
			},
			getter:          &mockGetter{},
			updater:         &mockUpdater{},
			strategy:        &mockStrategy{kind: "Deployment"},
			wantSuccessLen:  0,
			wantFailedLen:   0,
			wantErr:         true,
			wantErrContains: "error listing Deployments",
		},
		{
			name: "empty list returns empty slices",
			resource: &utils.K8sResource{
				NsList: []string{"default"},
				Period: &periodPkg.Period{},
			},
			lister: &mockLister{
				listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
					return []ResourceItem{}, nil
				},
			},
			getter:         &mockGetter{},
			updater:        &mockUpdater{},
			strategy:       &mockStrategy{kind: "Deployment"},
			wantSuccessLen: 0,
			wantFailedLen:  0,
			wantErr:        false,
		},
		{
			name: "name filter excludes non-matching items",
			resource: &utils.K8sResource{
				NsList: []string{"default"},
				Names:  []string{"app-a"},
				Period: &periodPkg.Period{},
			},
			lister: &mockLister{
				listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
					return []ResourceItem{
						newItem("app-a", "default"),
						newItem("app-b", "default"),
						newItem("app-c", "default"),
					}, nil
				},
			},
			getter: &mockGetter{
				getFn: func(_ context.Context, _, name string, _ metaV1.GetOptions) (ResourceItem, error) {
					return newItem(name, "default"), nil
				},
			},
			updater: &mockUpdater{
				updateFn: func(_ context.Context, _ string, r ResourceItem, _ metaV1.UpdateOptions) (ResourceItem, error) {
					return r, nil
				},
			},
			strategy: &mockStrategy{
				kind: "Deployment",
				applyScalingFn: func(_ context.Context, _ ResourceItem, _ string, _ *periodPkg.Period) (bool, error) {
					return false, nil
				},
			},
			wantSuccessLen: 1,
			wantFailedLen:  0,
			wantErr:        false,
		},
		{
			name: "getter non-context error adds to failed and continues",
			resource: &utils.K8sResource{
				NsList: []string{"default"},
				Period: &periodPkg.Period{},
			},
			lister: &mockLister{
				listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
					return []ResourceItem{
						newItem("fail-get", "default"),
						newItem("ok", "default"),
					}, nil
				},
			},
			getter: &mockGetter{
				getFn: func(_ context.Context, _, name string, _ metaV1.GetOptions) (ResourceItem, error) {
					if name == "fail-get" {
						return nil, errTest
					}

					return newItem(name, "default"), nil
				},
			},
			updater: &mockUpdater{
				updateFn: func(_ context.Context, _ string, r ResourceItem, _ metaV1.UpdateOptions) (ResourceItem, error) {
					return r, nil
				},
			},
			strategy: &mockStrategy{
				kind: "Deployment",
				applyScalingFn: func(_ context.Context, _ ResourceItem, _ string, _ *periodPkg.Period) (bool, error) {
					return false, nil
				},
			},
			wantSuccessLen: 1,
			wantFailedLen:  1,
			wantErr:        false,
		},
		{
			name: "getter returns context.Canceled stops loop and returns error",
			resource: &utils.K8sResource{
				NsList: []string{"default"},
				Period: &periodPkg.Period{},
			},
			lister: &mockLister{
				listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
					return []ResourceItem{
						newItem("item-1", "default"),
						newItem("item-2", "default"),
					}, nil
				},
			},
			getter: &mockGetter{
				getFn: func(_ context.Context, _, _ string, _ metaV1.GetOptions) (ResourceItem, error) {
					return nil, context.Canceled
				},
			},
			updater:        &mockUpdater{},
			strategy:       &mockStrategy{kind: "Deployment"},
			wantSuccessLen: 0,
			wantFailedLen:  1,
			wantErr:        true,
		},
		{
			name: "strategy ApplyScaling error adds to failed and continues",
			resource: &utils.K8sResource{
				NsList: []string{"default"},
				Period: &periodPkg.Period{},
			},
			lister: &mockLister{
				listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
					return []ResourceItem{
						newItem("fail-scale", "default"),
						newItem("ok", "default"),
					}, nil
				},
			},
			getter: &mockGetter{
				getFn: func(_ context.Context, _, name string, _ metaV1.GetOptions) (ResourceItem, error) {
					return newItem(name, "default"), nil
				},
			},
			updater: &mockUpdater{
				updateFn: func(_ context.Context, _ string, r ResourceItem, _ metaV1.UpdateOptions) (ResourceItem, error) {
					return r, nil
				},
			},
			strategy: &mockStrategy{
				kind: "Deployment",
				applyScalingFn: func(_ context.Context, r ResourceItem, _ string, _ *periodPkg.Period) (bool, error) {
					if r.GetName() == "fail-scale" {
						return false, errTest
					}

					return false, nil
				},
			},
			wantSuccessLen: 1,
			wantFailedLen:  1,
			wantErr:        false,
		},
		{
			name: "alreadyRestored true skips update and does not add to success",
			resource: &utils.K8sResource{
				NsList: []string{"default"},
				Period: &periodPkg.Period{},
			},
			lister: &mockLister{
				listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
					return []ResourceItem{newItem("restored", "default")}, nil
				},
			},
			getter: &mockGetter{
				getFn: func(_ context.Context, _, name string, _ metaV1.GetOptions) (ResourceItem, error) {
					return newItem(name, "default"), nil
				},
			},
			updater: &mockUpdater{
				updateFn: func(_ context.Context, _ string, _ ResourceItem, _ metaV1.UpdateOptions) (ResourceItem, error) {
					t.Fatal("updater should not be called when alreadyRestored is true")

					return nil, nil
				},
			},
			strategy: &mockStrategy{
				kind: "Deployment",
				applyScalingFn: func(_ context.Context, _ ResourceItem, _ string, _ *periodPkg.Period) (bool, error) {
					return true, nil
				},
			},
			wantSuccessLen: 0,
			wantFailedLen:  0,
			wantErr:        false,
		},
		{
			name: "updater error adds to failed",
			resource: &utils.K8sResource{
				NsList: []string{"default"},
				Period: &periodPkg.Period{},
			},
			lister: &mockLister{
				listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
					return []ResourceItem{newItem("fail-update", "default")}, nil
				},
			},
			getter: &mockGetter{
				getFn: func(_ context.Context, _, name string, _ metaV1.GetOptions) (ResourceItem, error) {
					return newItem(name, "default"), nil
				},
			},
			updater: &mockUpdater{
				updateFn: func(_ context.Context, _ string, _ ResourceItem, _ metaV1.UpdateOptions) (ResourceItem, error) {
					return nil, errTest
				},
			},
			strategy: &mockStrategy{
				kind: "Deployment",
				applyScalingFn: func(_ context.Context, _ ResourceItem, _ string, _ *periodPkg.Period) (bool, error) {
					return false, nil
				},
			},
			wantSuccessLen: 0,
			wantFailedLen:  1,
			wantErr:        false,
		},
		{
			name: "full success path",
			resource: &utils.K8sResource{
				NsList: []string{"default", "production"},
				Period: &periodPkg.Period{},
			},
			lister: &mockLister{
				listFn: func(_ context.Context, ns string, _ metaV1.ListOptions) ([]ResourceItem, error) {
					if ns == "default" {
						return []ResourceItem{newItem("app-1", "default")}, nil
					}

					return []ResourceItem{newItem("app-2", "production")}, nil
				},
			},
			getter: &mockGetter{
				getFn: func(_ context.Context, ns, name string, _ metaV1.GetOptions) (ResourceItem, error) {
					return newItem(name, ns), nil
				},
			},
			updater: &mockUpdater{
				updateFn: func(_ context.Context, _ string, r ResourceItem, _ metaV1.UpdateOptions) (ResourceItem, error) {
					return r, nil
				},
			},
			strategy: &mockStrategy{
				kind: "Deployment",
				applyScalingFn: func(_ context.Context, _ ResourceItem, _ string, _ *periodPkg.Period) (bool, error) {
					return false, nil
				},
			},
			wantSuccessLen: 2,
			wantFailedLen:  0,
			wantErr:        false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			processor := newTestProcessor(tc.lister, tc.getter, tc.updater, tc.strategy, tc.resource)

			success, failed, err := processor.ProcessResources(context.Background())

			if tc.wantErr {
				require.Error(t, err)

				if tc.wantErrContains != "" {
					assert.Contains(t, err.Error(), tc.wantErrContains)
				}
			} else {
				require.NoError(t, err)
			}

			assert.Len(t, success, tc.wantSuccessLen)
			assert.Len(t, failed, tc.wantFailedLen)
		})
	}
}

func TestProcessResources_ContextDeadlineExceeded(t *testing.T) {
	t.Parallel()

	resource := &utils.K8sResource{
		NsList: []string{"default"},
		Period: &periodPkg.Period{},
	}

	lister := &mockLister{
		listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
			return []ResourceItem{newItem("item-1", "default")}, nil
		},
	}

	getter := &mockGetter{
		getFn: func(_ context.Context, _, _ string, _ metaV1.GetOptions) (ResourceItem, error) {
			return nil, context.DeadlineExceeded
		},
	}

	strategy := &mockStrategy{kind: "StatefulSet"}
	processor := newTestProcessor(lister, getter, &mockUpdater{}, strategy, resource)

	_, _, err := processor.ProcessResources(context.Background())

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}

func TestProcessResources_FailedEntryContainsKindAndReason(t *testing.T) {
	t.Parallel()

	resource := &utils.K8sResource{
		NsList: []string{"default"},
		Period: &periodPkg.Period{},
	}

	lister := &mockLister{
		listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
			return []ResourceItem{newItem("my-deploy", "default")}, nil
		},
	}

	getter := &mockGetter{
		getFn: func(_ context.Context, _, _ string, _ metaV1.GetOptions) (ResourceItem, error) {
			return nil, errors.New("not found")
		},
	}

	strategy := &mockStrategy{kind: "CronJob"}
	processor := newTestProcessor(lister, getter, &mockUpdater{}, strategy, resource)

	_, failed, _ := processor.ProcessResources(context.Background())

	require.Len(t, failed, 1)
	assert.Equal(t, "CronJob", failed[0].Kind)
	assert.Equal(t, "my-deploy", failed[0].Name)
	assert.Equal(t, "not found", failed[0].Reason)
}

func TestProcessResources_SuccessEntryContainsKindAndName(t *testing.T) {
	t.Parallel()

	resource := &utils.K8sResource{
		NsList: []string{"default"},
		Period: &periodPkg.Period{},
	}

	lister := &mockLister{
		listFn: func(_ context.Context, _ string, _ metaV1.ListOptions) ([]ResourceItem, error) {
			return []ResourceItem{newItem("my-app", "default")}, nil
		},
	}

	getter := &mockGetter{
		getFn: func(_ context.Context, _, name string, _ metaV1.GetOptions) (ResourceItem, error) {
			return newItem(name, "default"), nil
		},
	}

	updater := &mockUpdater{
		updateFn: func(_ context.Context, _ string, r ResourceItem, _ metaV1.UpdateOptions) (ResourceItem, error) {
			return r, nil
		},
	}

	strategy := &mockStrategy{
		kind: "HPA",
		applyScalingFn: func(_ context.Context, _ ResourceItem, _ string, _ *periodPkg.Period) (bool, error) {
			return false, nil
		},
	}

	processor := newTestProcessor(lister, getter, updater, strategy, resource)

	success, _, err := processor.ProcessResources(context.Background())

	require.NoError(t, err)
	require.Len(t, success, 1)
	assert.Equal(t, "HPA", success[0].Kind)
	assert.Equal(t, "my-app", success[0].Name)
}
