package emitterir

var RewriteFeatureKey ExtensionFeatureKey = "Rewrite"

var _ ExtensionFeatureIR = &RewriteFeatureIR{}

type RewriteFeatureIR struct {
	ExtensionFeatureMetadata
	Target string
}

func (r *RewriteFeatureIR) Equals(other ExtensionFeatureIR) bool {
	rewrite, ok := other.(*RewriteFeatureIR)
	if !ok {
		return false
	}
	if r.Target != rewrite.Target {
		return false
	}
	return true
}
