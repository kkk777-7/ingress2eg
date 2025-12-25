package emitterir

var RateLimitFeatureKey ExtensionFeatureKey = "RateLimit"

var _ ExtensionFeatureIR = &RateLimitFeatureIR{}

type TimeUnit string

const (
	SecondTimeUnit TimeUnit = "Second"
	MinuteTimeUnit TimeUnit = "Minute"
)

type RateLimitFeatureIR struct {
	ExtensionFeatureMetadata
	LimitValue uint
	Unit       TimeUnit
}

func (r *RateLimitFeatureIR) Equals(other ExtensionFeatureIR) bool {
	rateLimit, ok := other.(*RateLimitFeatureIR)
	if !ok {
		return false
	}
	if r.LimitValue != rateLimit.LimitValue {
		return false
	}
	if r.Unit != rateLimit.Unit {
		return false
	}
	return true
}
