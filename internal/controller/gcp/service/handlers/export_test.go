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

package handlers

import gcpUtils "github.com/kubecloudscaler/kubecloudscaler/pkg/gcp/utils"

// Test-only exports compiled only into the test binary.

// ClientCloserForTest is the signature of the cache's Close seam, re-exported so tests
// can observe close-on-rotation deterministically.
type ClientCloserForTest func(cs *gcpUtils.ClientSet) error

// WithClientCloserForTest exposes the unexported withClientCloser option.
func WithClientCloserForTest(fn ClientCloserForTest) AuthHandlerOption {
	return withClientCloser(clientCloser(fn))
}
