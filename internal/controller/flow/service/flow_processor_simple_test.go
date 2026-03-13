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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rs/zerolog"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/service/testutil"
	"github.com/kubecloudscaler/kubecloudscaler/internal/controller/flow/types"
)

var _ = Describe("FlowProcessorService", func() {
	var (
		logger              zerolog.Logger
		mockValidator       *testutil.MockFlowValidator
		mockResourceMapper  *testutil.MockResourceMapper
		mockResourceCreator *testutil.MockResourceCreator
		svc                 *FlowProcessorService
	)

	BeforeEach(func() {
		logger = zerolog.Nop()
		mockValidator = &testutil.MockFlowValidator{}
		mockResourceMapper = &testutil.MockResourceMapper{}
		mockResourceCreator = &testutil.MockResourceCreator{}
		svc = NewFlowProcessorService(mockValidator, mockResourceMapper, mockResourceCreator, &logger)
	})

	Describe("ProcessFlow", func() {
		Context("when processing succeeds", func() {
			It("should process the flow without error", func() {
				flow := &kubecloudscalerv1alpha3.Flow{
					Spec: kubecloudscalerv1alpha3.FlowSpec{
						Flows: []kubecloudscalerv1alpha3.Flows{
							{
								PeriodName: "test-period",
								Resources: []kubecloudscalerv1alpha3.FlowResource{
									{Name: "test-resource"},
								},
							},
						},
					},
				}

				mockValidator.ExtractFlowDataFunc = func(
					f *kubecloudscalerv1alpha3.Flow,
				) (map[string]bool, map[string]bool, error) {
					return map[string]bool{"test-resource": true}, map[string]bool{"test-period": true}, nil
				}
				mockValidator.ValidatePeriodTimingsFunc = func(
					f *kubecloudscalerv1alpha3.Flow, periodNames map[string]bool,
				) error {
					return nil
				}
				mockResourceMapper.CreateResourceMappingsFunc = func(
					f *kubecloudscalerv1alpha3.Flow, resourceNames map[string]bool,
				) (map[string]types.ResourceInfo, error) {
					k8sRes := kubecloudscalerv1alpha3.K8sResource{Name: "test-resource"}
					return map[string]types.ResourceInfo{
						"test-resource": {
							Type:    "k8s",
							K8sRes:  &k8sRes,
							Periods: []types.PeriodWithDelay{},
						},
					}, nil
				}
				mockResourceCreator.CreateK8sResourceFunc = func(
					ctx context.Context,
					f *kubecloudscalerv1alpha3.Flow,
					resourceName string,
					k8sResource kubecloudscalerv1alpha3.K8sResource,
					periodsWithDelay []types.PeriodWithDelay,
				) error {
					return nil
				}

				err := svc.ProcessFlow(context.Background(), flow)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when extract flow data fails", func() {
			It("should return an error containing the failure context", func() {
				flow := &kubecloudscalerv1alpha3.Flow{}

				mockValidator.ExtractFlowDataFunc = func(
					f *kubecloudscalerv1alpha3.Flow,
				) (map[string]bool, map[string]bool, error) {
					return nil, nil, errors.New("extract error")
				}

				err := svc.ProcessFlow(context.Background(), flow)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to extract flow data"))
			})
		})
	})
})
