/*
Copyright 2024 The Kubernetes Authors.

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

package providerir

import (
	"fmt"
	"maps"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// MergeIRs accepts multiple IRs and creates a unique IR struct built
// as follows:
//   - GatewayClasses, Routes, and ReferenceGrants are grouped into the same maps
//   - Gateways may have the same NamespaceName even if they come from different
//     ingresses, as they have a their GatewayClass' name as name. For this reason,
//     if there are mutiple gateways named the same, their listeners are merged into
//     a unique Gateway.
//
// This behavior is likely to change after https://github.com/kubernetes-sigs/gateway-api/pull/1863 takes place.
func MergeIRs(irs ...ProviderIR) (ProviderIR, field.ErrorList) {
	mergedIRs := ProviderIR{
		Gateways:           make(map[types.NamespacedName]GatewayContext),
		GatewayClasses:     make(map[types.NamespacedName]gwapiv1.GatewayClass),
		HTTPRoutes:         make(map[types.NamespacedName]HTTPRouteContext),
		TLSRoutes:          make(map[types.NamespacedName]gwapiv1a2.TLSRoute),
		TCPRoutes:          make(map[types.NamespacedName]gwapiv1a2.TCPRoute),
		UDPRoutes:          make(map[types.NamespacedName]gwapiv1a2.UDPRoute),
		GRPCRoutes:         make(map[types.NamespacedName]gwapiv1.GRPCRoute),
		BackendTLSPolicies: make(map[types.NamespacedName]gwapiv1.BackendTLSPolicy),
		ReferenceGrants:    make(map[types.NamespacedName]gwapiv1b1.ReferenceGrant),
	}
	var errs field.ErrorList
	mergedIRs.Gateways, errs = mergeGatewayContexts(irs)
	if len(errs) > 0 {
		return ProviderIR{}, errs
	}
	// TODO(issue #189): Perform merge on HTTPRoute and Service like Gateway.
	for _, gr := range irs {
		maps.Copy(mergedIRs.GatewayClasses, gr.GatewayClasses)
		maps.Copy(mergedIRs.HTTPRoutes, gr.HTTPRoutes)
		maps.Copy(mergedIRs.TLSRoutes, gr.TLSRoutes)
		maps.Copy(mergedIRs.TCPRoutes, gr.TCPRoutes)
		maps.Copy(mergedIRs.UDPRoutes, gr.UDPRoutes)
		maps.Copy(mergedIRs.GRPCRoutes, gr.GRPCRoutes)
		maps.Copy(mergedIRs.BackendTLSPolicies, gr.BackendTLSPolicies)
		maps.Copy(mergedIRs.ReferenceGrants, gr.ReferenceGrants)
	}
	return mergedIRs, errs
}

func mergeGatewayContexts(irs []ProviderIR) (map[types.NamespacedName]GatewayContext, field.ErrorList) {
	newGatewayContexts := make(map[types.NamespacedName]GatewayContext)
	errs := field.ErrorList{}

	for _, currentIR := range irs {
		for _, g := range currentIR.Gateways {
			nn := types.NamespacedName{Namespace: g.Namespace, Name: g.Name}
			if existingGatewayContext, ok := newGatewayContexts[nn]; ok {
				g.Spec.Listeners = append(g.Spec.Listeners, existingGatewayContext.Spec.Listeners...)
				g.Spec.Addresses = append(g.Spec.Addresses, existingGatewayContext.Spec.Addresses...)
			}
			newGatewayContexts[nn] = GatewayContext{Gateway: g.Gateway}
			// 64 is the maximum number of listeners a Gateway can have
			if len(g.Spec.Listeners) > 64 {
				fieldPath := field.NewPath(fmt.Sprintf("%s/%s", nn.Namespace, nn.Name)).Child("spec").Child("listeners")
				errs = append(errs, field.Invalid(fieldPath, g, "error while merging gateway listeners: a gateway cannot have more than 64 listeners"))
			}
			// 16 is the maximum number of addresses a Gateway can have
			if len(g.Spec.Addresses) > 16 {
				fieldPath := field.NewPath(fmt.Sprintf("%s/%s", nn.Namespace, nn.Name)).Child("spec").Child("addresses")
				errs = append(errs, field.Invalid(fieldPath, g, "error while merging gateway listeners: a gateway cannot have more than 16 addresses"))
			}
		}
	}
	return newGatewayContexts, errs
}
