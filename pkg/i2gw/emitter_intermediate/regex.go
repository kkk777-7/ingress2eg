package emitterir

var RegexFeatureKey ExtensionFeatureKey = "Regex"

var _ ExtensionFeatureIR = &RegexFeatureIR{}

type RegexFeatureIR struct {
	ExtensionFeatureMetadata
	PathPattern string
}

func (r *RegexFeatureIR) Equals(other ExtensionFeatureIR) bool {
	regex, ok := other.(*RegexFeatureIR)
	if !ok {
		return false
	}
	if r.PathPattern != regex.PathPattern {
		return false
	}
	return true
}
