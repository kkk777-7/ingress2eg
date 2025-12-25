package emitterir

import "slices"

var IpRangeFeatureKey ExtensionFeatureKey = "IPRange"

var _ ExtensionFeatureIR = &IpRangeFeatureIR{}

type IpRangeFeatureIR struct {
	ExtensionFeatureMetadata
	AllowList []string
	DenyList  []string
}

func (r *IpRangeFeatureIR) Equals(other ExtensionFeatureIR) bool {
	redirect, ok := other.(*IpRangeFeatureIR)
	if !ok {
		return false
	}
	al1 := slices.Clone(r.AllowList)
	al2 := slices.Clone(redirect.AllowList)
	dl1 := slices.Clone(r.DenyList)
	dl2 := slices.Clone(redirect.DenyList)
	slices.Sort(al1)
	slices.Sort(al2)
	slices.Sort(dl1)
	slices.Sort(dl2)

	if !slices.Equal(al1, al2) {
		return false
	}
	return slices.Equal(dl1, dl2)
}
