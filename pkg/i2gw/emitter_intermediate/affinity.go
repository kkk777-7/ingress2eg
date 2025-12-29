package emitterir

import gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

var AffinityFeatureKey ExtensionFeatureKey = "Affinity"

var _ ExtensionFeatureIR = &AffinityFeatureIR{}

type AffinityFeatureIR struct {
	ExtensionFeatureMetadata
	CookieName       string
	CookieSameSite   string
	CookieTTLSeconds *gwapiv1.Duration
}

func (r *AffinityFeatureIR) Equals(other ExtensionFeatureIR) bool {
	affinity, ok := other.(*AffinityFeatureIR)
	if !ok {
		return false
	}
	if r.CookieName != affinity.CookieName {
		return false
	}
	if r.CookieSameSite != affinity.CookieSameSite {
		return false
	}
	if (r.CookieTTLSeconds == nil) != (affinity.CookieTTLSeconds == nil) {
		return false
	}
	if r.CookieTTLSeconds != nil && *r.CookieTTLSeconds != *affinity.CookieTTLSeconds {
		return false
	}
	return true
}
