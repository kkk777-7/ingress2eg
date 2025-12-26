package emitterir

var BufferFeatureKey ExtensionFeatureKey = "Buffer"

var _ ExtensionFeatureIR = &BufferFeatureIR{}

type BufferFeatureIR struct {
	ExtensionFeatureMetadata
	LimitValue string
}

func (r *BufferFeatureIR) Equals(other ExtensionFeatureIR) bool {
	Buffer, ok := other.(*BufferFeatureIR)
	if !ok {
		return false
	}
	if r.LimitValue != Buffer.LimitValue {
		return false
	}
	return true
}
