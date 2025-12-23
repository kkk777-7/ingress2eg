package emitterir

var RegexFeatureKey ExtensionFeatureKey = "Regex"

var _ ExtensionFeatureIR = &RegexFeatureIR{}

type RegexFeatureIR struct {
	ExtensionFeatureMetadata
}

func (r *RegexFeatureIR) Equals(other ExtensionFeatureIR) bool {
	return true
}
