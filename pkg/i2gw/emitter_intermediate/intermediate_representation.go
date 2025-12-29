/*
Copyright 2025 The Kubernetes Authors.

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

package emitterir

import (
	"k8s.io/apimachinery/pkg/types"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

const (
	ListenerAllIndex  = -1
	RouteRuleAllIndex = -1
)

// EmitterIR holds specifications of Gateway Objects for supporting Ingress extensions,
// annotations, and proprietary API features not supported as Gateway core
// features. An EmitterIR field can be mapped to core Gateway-API fields,
// or provider-specific Gateway extensions.
type EmitterIR struct {
	Gateways   map[types.NamespacedName]GatewayContext
	HTTPRoutes map[types.NamespacedName]HTTPRouteContext

	GatewayClasses map[types.NamespacedName]GatewayClassContext
	TLSRoutes      map[types.NamespacedName]TLSRouteContext
	TCPRoutes      map[types.NamespacedName]TCPRouteContext
	UDPRoutes      map[types.NamespacedName]UDPRouteContext
	GRPCRoutes     map[types.NamespacedName]GRPCRouteContext

	BackendTLSPolicies map[types.NamespacedName]BackendTLSPolicyContext
	ReferenceGrants    map[types.NamespacedName]ReferenceGrantContext
}

type GatewayContext struct {
	gwapiv1.Gateway

	// ExtensionFeatures maps extension feature keys to listener-specific features.
	// The inner map key is the listener index. Use ListenerAllIndex (-1) for gateway-level features.
	ExtensionFeatures map[ExtensionFeatureKey]map[int]ExtensionFeatureIR
}

// MergeExtensionFeature merges extension features across all listeners into a gateway-level feature
// if all listeners have identical extension features for the given key.
// This consolidates redundant listener-level features into a single gateway-level feature.
func (g *GatewayContext) MergeExtensionFeature(key ExtensionFeatureKey) {
	if g.ExtensionFeatures == nil {
		return
	}
	efMap, ok := g.ExtensionFeatures[key]
	if !ok {
		return
	}
	if len(efMap) != len(g.Spec.Listeners) {
		return
	}

	var first *ExtensionFeatureIR
	for _, feature := range efMap {
		if first == nil {
			first = &feature
			continue
		}
		if !(*first).Equals(feature) {
			return
		}
	}

	g.ExtensionFeatures[key] = map[int]ExtensionFeatureIR{ListenerAllIndex: *first}
}

type HTTPRouteContext struct {
	gwapiv1.HTTPRoute

	// ExtensionFeatures maps extension feature keys to rule-specific features.
	// The inner map key is the rule index. Use RouteRuleAllIndex (-1) for route-level features.
	ExtensionFeatures map[ExtensionFeatureKey]map[int]ExtensionFeatureIR
}

// MergeExtensionFeature merges extension features across all rules into a route-level feature
// if all rules have identical extension features for the given key.
// This consolidates redundant rule-level features into a single route-level feature.
func (h *HTTPRouteContext) MergeExtensionFeature(key ExtensionFeatureKey) {
	if h.ExtensionFeatures == nil {
		return
	}
	efMap, ok := h.ExtensionFeatures[key]
	if !ok {
		return
	}
	if len(efMap) != len(h.Spec.Rules) {
		return
	}

	var first *ExtensionFeatureIR
	for _, feature := range efMap {
		if first == nil {
			first = &feature
			continue
		}
		if !(*first).Equals(feature) {
			return
		}
	}

	h.ExtensionFeatures[key] = map[int]ExtensionFeatureIR{RouteRuleAllIndex: *first}
}

type GatewayClassContext struct {
	gwapiv1.GatewayClass
}

type TLSRouteContext struct {
	gwapiv1a2.TLSRoute
}

type TCPRouteContext struct {
	gwapiv1a2.TCPRoute
}

type UDPRouteContext struct {
	gwapiv1a2.UDPRoute
}

type GRPCRouteContext struct {
	gwapiv1.GRPCRoute

	// ExtensionFeatures maps extension feature keys to rule-specific features.
	// The inner map key is the rule index. Use RouteRuleAllIndex (-1) for route-level features.
	ExtensionFeatures map[ExtensionFeatureKey]map[int]ExtensionFeatureIR
}

// MergeExtensionFeature merges extension features across all rules into a route-level feature
// if all rules have identical extension features for the given key.
// This consolidates redundant rule-level features into a single route-level feature.
func (g *GRPCRouteContext) MergeExtensionFeature(key ExtensionFeatureKey) {
	if g.ExtensionFeatures == nil {
		return
	}
	efMap, ok := g.ExtensionFeatures[key]
	if !ok {
		return
	}
	if len(efMap) != len(g.Spec.Rules) {
		return
	}

	var first *ExtensionFeatureIR
	for _, feature := range efMap {
		if first == nil {
			first = &feature
			continue
		}
		if !(*first).Equals(feature) {
			return
		}
	}
	g.ExtensionFeatures[key] = map[int]ExtensionFeatureIR{RouteRuleAllIndex: *first}
}

type BackendTLSPolicyContext struct {
	gwapiv1.BackendTLSPolicy
}

type ReferenceGrantContext struct {
	gwapiv1b1.ReferenceGrant
}
