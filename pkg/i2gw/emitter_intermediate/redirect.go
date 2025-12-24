package emitterir

var RedirectFeatureKey ExtensionFeatureKey = "Redirect"

var _ ExtensionFeatureIR = &RedirectFeatureIR{}

type RedirectFeatureIR struct {
	ExtensionFeatureMetadata
	Url        string
	StatusCode int
	Ssl        *SslRefirectIR
}

type SslRefirectIR struct {
	Force bool
}

func (r *RedirectFeatureIR) Equals(other ExtensionFeatureIR) bool {
	redirect, ok := other.(*RedirectFeatureIR)
	if !ok {
		return false
	}
	if r.Url != redirect.Url {
		return false
	}
	if r.StatusCode != redirect.StatusCode {
		return false
	}
	if (r.Ssl == nil) != (redirect.Ssl == nil) {
		return false
	}
	if r.Ssl != nil && redirect.Ssl != nil {
		if r.Ssl.Force != redirect.Ssl.Force {
			return false
		}
	}
	return true
}
