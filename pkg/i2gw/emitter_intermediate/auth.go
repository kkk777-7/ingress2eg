package emitterir

import "slices"

var (
	BasicAuthFeatureKey    ExtensionFeatureKey = "BasicAuth"
	MTLSFeatureKey         ExtensionFeatureKey = "mTLS"
	ExternalAuthFeatureKey ExtensionFeatureKey = "ExternalAuth"
)

var _ ExtensionFeatureIR = &BasicAuthFeatureIR{}

type BasicAuthFeatureIR struct {
	ExtensionFeatureMetadata
	SecretReference
}

type SecretReference struct {
	Name      string
	Namespace string
}

func (r *BasicAuthFeatureIR) Equals(other ExtensionFeatureIR) bool {
	auth, ok := other.(*BasicAuthFeatureIR)
	if !ok {
		return false
	}
	if r.Name != auth.Name {
		return false
	}
	if r.Namespace != auth.Namespace {
		return false
	}
	return true
}

var _ ExtensionFeatureIR = &MTLSFeatureIR{}

type MTLSFeatureIR struct {
	ExtensionFeatureMetadata
	SecretReference
}

func (r *MTLSFeatureIR) Equals(other ExtensionFeatureIR) bool {
	mtls, ok := other.(*MTLSFeatureIR)
	if !ok {
		return false
	}
	if r.Name != mtls.Name {
		return false
	}
	if r.Namespace != mtls.Namespace {
		return false
	}
	return true
}

var _ ExtensionFeatureIR = &ExternalAuthFeatureIR{}

type ExternalAuthFeatureIR struct {
	ExtensionFeatureMetadata
	Url                    string
	AllowedResponseHeaders []string
}

func (r *ExternalAuthFeatureIR) Equals(other ExtensionFeatureIR) bool {
	extAuth, ok := other.(*ExternalAuthFeatureIR)
	if !ok {
		return false
	}
	if r.Url != extAuth.Url {
		return false
	}
	if len(r.AllowedResponseHeaders) != len(extAuth.AllowedResponseHeaders) {
		return false
	}
	// Sort and compare to ignore order
	r1 := slices.Clone(r.AllowedResponseHeaders)
	r2 := slices.Clone(extAuth.AllowedResponseHeaders)
	slices.Sort(r1)
	slices.Sort(r2)
	return slices.Equal(r1, r2)
}
