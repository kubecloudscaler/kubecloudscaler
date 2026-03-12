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
	"time"

	"github.com/rs/zerolog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubecloudscalerv1alpha3 "github.com/kubecloudscaler/kubecloudscaler/api/v1alpha3"
)

// FlowReconciliationContext holds the shared state for the Flow Chain of Responsibility.
// Each handler can read from and write to this context.
//
// Fields are populated by handlers as the chain executes:
//   - Initial: Ctx, Request, Client, Logger set by controller
//   - After Fetch: Flow set
//   - After Finalizer: SkipRemaining may be set (deletion case)
//   - After Processing: Flow resources created/updated
//   - After Status: Status updated in cluster
type FlowReconciliationContext struct {
	// Ctx is the Go context from the Reconcile method.
	// Set by: Controller (before chain execution)
	Ctx context.Context

	// Request is the controller request with NamespacedName.
	// Set by: Controller (before chain execution)
	Request ctrl.Request

	// Client is the Kubernetes client for API operations.
	// Set by: Controller (before chain execution)
	Client client.Client

	// Logger is the structured logger for handler execution logging.
	// Set by: Controller (before chain execution)
	Logger *zerolog.Logger

	// Flow is the resource being reconciled.
	// Set by: FetchHandler
	Flow *kubecloudscalerv1alpha3.Flow

	// SkipRemaining stops the chain early (e.g., during deletion cleanup).
	// Set by: Any handler (e.g., FinalizerHandler on deletion)
	SkipRemaining bool

	// RequeueAfter is the requeue delay duration (first handler to set wins).
	// Set by: Any handler
	// Used by: Controller (uses this value in ctrl.Result)
	RequeueAfter time.Duration
}
