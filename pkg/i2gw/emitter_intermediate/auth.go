package emitterir

var BasicAuthFeatureKey ExtensionFeatureKey = "BasicAuth"
var MTLSFeatureKey ExtensionFeatureKey = "mTLS"

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
