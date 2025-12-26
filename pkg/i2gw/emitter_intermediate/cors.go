package emitterir

import (
	"slices"

	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var CORSFeatureKey ExtensionFeatureKey = "CORS"

var _ ExtensionFeatureIR = &CORSFeatureIR{}

type CORSFeatureIR struct {
	ExtensionFeatureMetadata
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	MaxAge           *gwapiv1.Duration
	AllowCredentials *bool
}

func (r *CORSFeatureIR) Equals(other ExtensionFeatureIR) bool {
	cors, ok := other.(*CORSFeatureIR)
	if !ok {
		return false
	}

	if !equalStringSlicesIgnoreOrder(r.AllowOrigins, cors.AllowOrigins) {
		return false
	}
	if !equalStringSlicesIgnoreOrder(r.AllowMethods, cors.AllowMethods) {
		return false
	}
	if !equalStringSlicesIgnoreOrder(r.AllowHeaders, cors.AllowHeaders) {
		return false
	}
	if !equalStringSlicesIgnoreOrder(r.ExposeHeaders, cors.ExposeHeaders) {
		return false
	}

	if (r.MaxAge == nil) != (cors.MaxAge == nil) {
		return false
	}
	if r.MaxAge != nil && *r.MaxAge != *cors.MaxAge {
		return false
	}

	if (r.AllowCredentials == nil) != (cors.AllowCredentials == nil) {
		return false
	}
	if r.AllowCredentials != nil && *r.AllowCredentials != *cors.AllowCredentials {
		return false
	}
	return true
}

// ignore order
func equalStringSlicesIgnoreOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aCopy := slices.Clone(a)
	bCopy := slices.Clone(b)
	slices.Sort(aCopy)
	slices.Sort(bCopy)

	return slices.Equal(aCopy, bCopy)
}
