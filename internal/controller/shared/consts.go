package shared

import "time"

const (
	// ScalerFinalizer is the finalizer name shared by K8s and GCP scaler resources.
	ScalerFinalizer = "kubecloudscaler.cloud/finalizer"

	// TransientRequeueAfter is the default requeue duration for transient errors in handlers.
	TransientRequeueAfter = 30 * time.Second
)
