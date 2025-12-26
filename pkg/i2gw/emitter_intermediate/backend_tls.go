package emitterir

import (
	"reflect"

	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var BackendTLSFeatureKey ExtensionFeatureKey = "BackendTLS"

var _ ExtensionFeatureIR = &BackendTLSFeatureIR{}

type BackendTLSFeatureIR struct {
	ExtensionFeatureMetadata
	CertificateRef     gwapiv1.LocalObjectReference
	Sni                string
	InsecureSkipVerify *bool
}

func (r *BackendTLSFeatureIR) Equals(other ExtensionFeatureIR) bool {
	backendtls, ok := other.(*BackendTLSFeatureIR)
	if !ok {
		return false
	}
	if r.Sni != backendtls.Sni {
		return false
	}
	if (r.InsecureSkipVerify == nil) != (backendtls.InsecureSkipVerify == nil) {
		return false
	}
	if r.InsecureSkipVerify != nil && *r.InsecureSkipVerify != *backendtls.InsecureSkipVerify {
		return false
	}
	return reflect.DeepEqual(r.CertificateRef, backendtls.CertificateRef)
}
