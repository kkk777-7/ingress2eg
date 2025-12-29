package emitterir

import "slices"

var RetryFeatureKey ExtensionFeatureKey = "Retry"

var _ ExtensionFeatureIR = &RetryFeatureIR{}

type RetryFeatureIR struct {
	ExtensionFeatureMetadata
	Triggers []string
	Count    *int32
}

func (r *RetryFeatureIR) Equals(other ExtensionFeatureIR) bool {
	retry, ok := other.(*RetryFeatureIR)
	if !ok {
		return false
	}
	if (r.Count == nil) != (retry.Count == nil) {
		return false
	}
	if r.Count != nil && *r.Count != *retry.Count {
		return false
	}
	rt1 := slices.Clone(r.Triggers)
	rt2 := slices.Clone(retry.Triggers)
	slices.Sort(rt1)
	slices.Sort(rt2)
	if !slices.Equal(rt1, rt2) {
		return false
	}
	return true
}
