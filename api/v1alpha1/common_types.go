// +kubebuilder:object:generate=true
package v1alpha1

type ScalerPeriod struct {
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Enum=down;up
	Type string     `json:"type"`
	Time TimePeriod `json:"time"`
	// Minimum replicas
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// Maximum replicas
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
}

type TimePeriod struct {
	Recurring *RecurringPeriod `json:"recurring,omitempty"`
	Fixed     *FixedPeriod     `json:"fixed,omitempty"`
}

type RecurringPeriod struct {
	Days []string `json:"days"`
	// +kubebuilder:validation:Pattern=`^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$`
	StartTime string `json:"startTime"`
	// TimeRef is a reference to a named time period in the chosen scaler.
	// The format is <scaler kind>/<scaler name>@<named scalerPeriod>
	// +kubebuilder:validation:Pattern=`^[\w\-]+\/[\w\-]+@[\w.\-_]+$`
	StartTimeRef *string `json:"startTimeRef,omitempty"`
	// +kubebuilder:validation:Pattern=`^\d*s$`
	StartGracePeriod *string `json:"startGracePeriod,omitempty"`
	// +kubebuilder:validation:Pattern=`^([0-1]?[0-9]|2[0-3]):[0-5][0-9]$`
	EndTime string `json:"endTime"`
	// TimeRef is a reference to a named time period in the chosen scaler.
	// The format is <scaler kind>/<scaler name>@<named scalerPeriod>
	// +kubebuilder:validation:Pattern=`^[\w\-]+\/[\w\-]+@[\w.\-_]+$`
	EndTimeRef *string `json:"endTimeRef,omitempty"`
	// +kubebuilder:validation:Pattern=`^\d*s$`
	EndGracePeriod *string `json:"endgracePeriod,omitempty"`
	Timezone       *string `json:"timezone,omitempty"`
	// Run once at StartTime
	Once *bool `json:"once,omitempty"`
	// Reverse the period
	Reverse *bool `json:"reverse,omitempty"`
}

type FixedPeriod struct {
	// +kubebuilder:validation:Pattern=`^\d{4}-(0?[1-9]|1[0,1,2])-(0?[1-9]|[12][0-9]|3[01]) ([0-1]?[0-9]|2[0-3]):[0-5]?[0-9]:[0-5]?[0-9]$`
	StartTime string `json:"startTime"`
	// +kubebuilder:validation:Pattern=`^\d{4}-(0?[1-9]|1[0,1,2])-(0?[1-9]|[12][0-9]|3[01]) ([0-1]?[0-9]|2[0-3]):[0-5]?[0-9]:[0-5]?[0-9]$`
	EndTime  string  `json:"endTime"`
	Timezone *string `json:"timezone,omitempty"`
	// Run once at StartTime
	Once *bool `json:"once,omitempty"`
	// Grace period in seconds for deployments before scaling down
	// +kubebuilder:validation:Pattern=`^\d*s$`
	StartGracePeriod *string `json:"startGracePeriod,omitempty"`
	// +kubebuilder:validation:Pattern=`^\d*s$`
	EndGracePeriod *string `json:"endGracePeriod,omitempty"`
	// Reverse the period
	Reverse *bool `json:"reverse,omitempty"`
}

// ScalerStatus defines the observed state of Scaler
type ScalerStatus struct {
	CurrentPeriod *ScalerStatusPeriod `json:"currentPeriod,omitempty"`
	Comments      *string             `json:"comments,omitempty"`
}

type ScalerStatusPeriod struct {
	Spec       *RecurringPeriod      `json:"spec"`
	SpecSHA    string                `json:"specSHA"`
	Successful []ScalerStatusSuccess `json:"success,omitempty"`
	Failed     []ScalerStatusFailed  `json:"failed,omitempty"`
}

type ScalerStatusSuccess struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type ScalerStatusFailed struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}
