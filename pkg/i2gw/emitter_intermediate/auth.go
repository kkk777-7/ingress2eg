package emitterir

var BasicAuthFeatureKey ExtensionFeatureKey = "BasicAuth"

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
