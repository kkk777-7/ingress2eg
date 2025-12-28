package emitterir

import gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

var TimeoutFeatureKey ExtensionFeatureKey = "Timeout"

var _ ExtensionFeatureIR = &TimeoutFeatureIR{}

type TimeoutFeatureIR struct {
	ExtensionFeatureMetadata
	Duration *gwapiv1.Duration
}

func (r *TimeoutFeatureIR) Equals(other ExtensionFeatureIR) bool {
	timeout, ok := other.(*TimeoutFeatureIR)
	if !ok {
		return false
	}
	if (r.Duration == nil) != (timeout.Duration == nil) {
		return false
	}
	if r.Duration != nil && *r.Duration != *timeout.Duration {
		return false
	}
	return true
}
