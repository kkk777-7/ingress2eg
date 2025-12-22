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

package utils

import (
	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

type uniqueBackendRefsKey struct {
	Name      gwapiv1.ObjectName
	Namespace gwapiv1.Namespace
	Port      gwapiv1.PortNumber
	Group     gwapiv1.Group
	Kind      gwapiv1.Kind
}

// removeBackendRefsDuplicates removes duplicate backendRefs from a list of backendRefs.
func removeBackendRefsDuplicates(backendRefs []gwapiv1.HTTPBackendRef) []gwapiv1.HTTPBackendRef {

	uniqueBackendRefs := map[uniqueBackendRefsKey]*gwapiv1.HTTPBackendRef{}

	for _, backendRef := range backendRefs {
		var k uniqueBackendRefsKey

		group := gwapiv1.Group("")
		kind := gwapiv1.Kind("Service")

		if backendRef.Group != nil && *backendRef.Group != "core" {
			group = *backendRef.Group
		}

		if backendRef.Kind != nil {
			kind = *backendRef.Kind
		}

		k.Name = backendRef.Name
		k.Group = group
		k.Kind = kind

		if backendRef.Port != nil {
			k.Port = *backendRef.Port
		}

		if oldRef, exists := uniqueBackendRefs[k]; exists {
			if oldRef.Weight != nil && backendRef.Weight != nil {
				*oldRef.Weight += *backendRef.Weight
			}
		} else {
			uniqueBackendRefs[k] = backendRef.DeepCopy()
		}
	}
	result := make([]gwapiv1.HTTPBackendRef, 0, len(uniqueBackendRefs))
	for _, backendRef := range uniqueBackendRefs {
		result = append(result, *backendRef)
	}
	return result
}

// ToGatewayResources converts the received emitterir.IR to i2gw.GatewayResource
// without taking into consideration any emitter specific logic.
func ToGatewayResources(ir emitterir.EmitterIR) (i2gw.GatewayResources, field.ErrorList) {
	gatewayResources := i2gw.GatewayResources{
		Gateways:           make(map[types.NamespacedName]gwapiv1.Gateway),
		HTTPRoutes:         make(map[types.NamespacedName]gwapiv1.HTTPRoute),
		GatewayClasses:     make(map[types.NamespacedName]gwapiv1.GatewayClass),
		GRPCRoutes:         make(map[types.NamespacedName]gwapiv1.GRPCRoute),
		TLSRoutes:          make(map[types.NamespacedName]gwapiv1a2.TLSRoute),
		TCPRoutes:          make(map[types.NamespacedName]gwapiv1a2.TCPRoute),
		UDPRoutes:          make(map[types.NamespacedName]gwapiv1a2.UDPRoute),
		BackendTLSPolicies: make(map[types.NamespacedName]gwapiv1.BackendTLSPolicy),
		ReferenceGrants:    make(map[types.NamespacedName]gwapiv1b1.ReferenceGrant),
	}
	for key, gatewayContext := range ir.Gateways {
		gatewayResources.Gateways[key] = gatewayContext.Gateway
	}
	for key, httpRouteContext := range ir.HTTPRoutes {
		gatewayResources.HTTPRoutes[key] = httpRouteContext.HTTPRoute
		hr := gatewayResources.HTTPRoutes[key]
		for i := range hr.Spec.Rules {
			hr.Spec.Rules[i].BackendRefs = removeBackendRefsDuplicates(hr.Spec.Rules[i].BackendRefs)
		}
		gatewayResources.HTTPRoutes[key] = hr
	}
	for key, val := range ir.GatewayClasses {
		gatewayResources.GatewayClasses[key] = val.GatewayClass
	}
	for key, val := range ir.GRPCRoutes {
		gatewayResources.GRPCRoutes[key] = val.GRPCRoute
	}
	for key, val := range ir.TLSRoutes {
		gatewayResources.TLSRoutes[key] = val.TLSRoute
	}
	for key, val := range ir.TCPRoutes {
		gatewayResources.TCPRoutes[key] = val.TCPRoute
	}
	for key, val := range ir.UDPRoutes {
		gatewayResources.UDPRoutes[key] = val.UDPRoute
	}
	for key, val := range ir.BackendTLSPolicies {
		gatewayResources.BackendTLSPolicies[key] = val.BackendTLSPolicy
	}
	for key, val := range ir.ReferenceGrants {
		gatewayResources.ReferenceGrants[key] = val.ReferenceGrant
	}
	return gatewayResources, nil
}
