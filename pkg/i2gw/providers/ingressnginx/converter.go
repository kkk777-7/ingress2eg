/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ingressnginx

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	provider_intermediate "github.com/kkk777-7/ingress2eg/pkg/i2gw/provider_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/providers/common"
)

// resourcesToIRConverter implements the ToIR function of i2gw.ResourcesToIRConverter interface.
type resourcesToIRConverter struct {
	featureParsers                []i2gw.FeatureParser
	implementationSpecificOptions i2gw.ProviderImplementationSpecificOptions
}

// newResourcesToIRConverter returns an ingress-nginx resourcesToIRConverter instance.
func newResourcesToIRConverter() *resourcesToIRConverter {
	return &resourcesToIRConverter{
		featureParsers: []i2gw.FeatureParser{
			canaryFeature,
			regexFeature,
			rewriteFeature,
			redirectFeature,
			authFeature,
			ipRangeFeature,
			rateLimitFeature,
			proxyBufferFeature,
			headerFeature,
			corsFeature,
			backendTLSFeature,
			timeOutFeature,
			retryFeature,
			affinityFeature,
			sslPassthroughFeature,
			backendProtocolFeature, // Must be after backendTLSFeature to copy TLS features
		},
		implementationSpecificOptions: i2gw.ProviderImplementationSpecificOptions{
			ToImplementationSpecificHTTPPathTypeMatch: implementationSpecificHTTPPathTypeMatch,
		},
	}
}

func (c *resourcesToIRConverter) convert(storage *storage) (emitterir.EmitterIR, field.ErrorList) {
	// TODO(liorliberman) temporary until we decide to change ToIR and featureParsers to get a map of [types.NamespacedName]*networkingv1.Ingress instead of a list
	ingressList := storage.Ingresses.List()

	// Convert plain ingress resources to gateway resources, ignoring all
	// provider-specific features.
	pir, errs := common.ToIR(ingressList, storage.ServicePorts, c.implementationSpecificOptions)
	if len(errs) > 0 {
		return emitterir.EmitterIR{}, errs
	}
	eir := provider_intermediate.ToEmitterIR(pir)

	for _, parseFeatureFunc := range c.featureParsers {
		// Apply the feature parsing function to the gateway resources, one by one.
		parseErrs := parseFeatureFunc(ingressList, storage.ServicePorts, &pir, &eir)
		// Append the parsing errors to the error list.
		errs = append(errs, parseErrs...)
	}
	return eir, errs
}
