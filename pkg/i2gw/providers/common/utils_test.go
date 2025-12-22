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

package common

import (
	"testing"

	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestGroupIngressPathsByMatchKey(t *testing.T) {
	iPrefix := networkingv1.PathTypePrefix

	testCases := []struct {
		name     string
		rules    []ingressRule
		expected orderedIngressPathsByMatchKey
	}{
		{
			name:  "no rules",
			rules: []ingressRule{},
			expected: orderedIngressPathsByMatchKey{
				keys: []pathMatchKey{},
				data: map[pathMatchKey][]ingressPath{},
			},
		},
		{
			name: "1 rule with 1 match",
			rules: []ingressRule{
				{
					rule: networkingv1.IngressRule{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/test",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test",
												Port: networkingv1.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: orderedIngressPathsByMatchKey{
				keys: []pathMatchKey{
					"Prefix//test",
				},
				data: map[pathMatchKey][]ingressPath{
					"Prefix//test": {
						{
							ruleIdx:  0,
							pathIdx:  0,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test",
										Port: networkingv1.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "1 rule, multiple matches, different path",
			rules: []ingressRule{
				{
					rule: networkingv1.IngressRule{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/test1",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test1",
												Port: networkingv1.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
									{
										Path:     "/test2",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test2",
												Port: networkingv1.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: orderedIngressPathsByMatchKey{
				keys: []pathMatchKey{
					"Prefix//test1",
					"Prefix//test2",
				},
				data: map[pathMatchKey][]ingressPath{
					"Prefix//test1": {
						{
							ruleIdx:  0,
							pathIdx:  0,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test1",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test1",
										Port: networkingv1.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
					"Prefix//test2": {
						{
							ruleIdx:  0,
							pathIdx:  1,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test2",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test2",
										Port: networkingv1.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple rules with single matches, same path",
			rules: []ingressRule{
				{
					rule: networkingv1.IngressRule{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/test",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test",
												Port: networkingv1.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
								},
							},
						},
					},
				},
				{
					rule: networkingv1.IngressRule{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/test",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test",
												Port: networkingv1.ServiceBackendPort{
													Number: 81,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: orderedIngressPathsByMatchKey{
				keys: []pathMatchKey{
					"Prefix//test",
				},
				data: map[pathMatchKey][]ingressPath{
					"Prefix//test": {
						{
							ruleIdx:  0,
							pathIdx:  0,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test",
										Port: networkingv1.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
						{
							ruleIdx:  1,
							pathIdx:  0,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test",
										Port: networkingv1.ServiceBackendPort{
											Number: 81,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple rules with single matches, different path",
			rules: []ingressRule{
				{
					rule: networkingv1.IngressRule{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/test",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test",
												Port: networkingv1.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
								},
							},
						},
					},
				},
				{
					rule: networkingv1.IngressRule{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/test2",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test2",
												Port: networkingv1.ServiceBackendPort{
													Number: 81,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: orderedIngressPathsByMatchKey{
				keys: []pathMatchKey{
					"Prefix//test",
					"Prefix//test2",
				},
				data: map[pathMatchKey][]ingressPath{
					"Prefix//test": {
						{
							ruleIdx:  0,
							pathIdx:  0,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test",
										Port: networkingv1.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
					"Prefix//test2": {
						{
							ruleIdx:  1,
							pathIdx:  0,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test2",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test2",
										Port: networkingv1.ServiceBackendPort{
											Number: 81,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple rules with multiple matches, mixed paths",
			rules: []ingressRule{
				{
					rule: networkingv1.IngressRule{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/test11",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test11",
												Port: networkingv1.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
									{
										Path:     "/test12",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test12",
												Port: networkingv1.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
								},
							},
						},
					},
				},
				{
					rule: networkingv1.IngressRule{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/test21",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test21",
												Port: networkingv1.ServiceBackendPort{
													Number: 81,
												},
											},
										},
									},
									{
										Path:     "/test11",
										PathType: ptr.To(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test11",
												Port: networkingv1.ServiceBackendPort{
													Number: 81,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: orderedIngressPathsByMatchKey{
				keys: []pathMatchKey{
					"Prefix//test11",
					"Prefix//test12",
					"Prefix//test21",
				},
				data: map[pathMatchKey][]ingressPath{
					"Prefix//test11": {
						{
							ruleIdx:  0,
							pathIdx:  0,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test11",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test11",
										Port: networkingv1.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
						{
							ruleIdx:  1,
							pathIdx:  1,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test11",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test11",
										Port: networkingv1.ServiceBackendPort{
											Number: 81,
										},
									},
								},
							},
						},
					},
					"Prefix//test12": {
						{
							ruleIdx:  0,
							pathIdx:  1,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test12",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test12",
										Port: networkingv1.ServiceBackendPort{
											Number: 80,
										},
									},
								},
							},
						},
					},
					"Prefix//test21": {
						{
							ruleIdx:  1,
							pathIdx:  0,
							ruleType: "http",
							path: networkingv1.HTTPIngressPath{
								Path:     "/test21",
								PathType: &iPrefix,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test21",
										Port: networkingv1.ServiceBackendPort{
											Number: 81,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, groupIngressPathsByMatchKey(tc.rules))
		})
	}
}

func TestGroupServicePortsByPortName(t *testing.T) {
	t.Run("group service ports by port name", func(t *testing.T) {
		services := map[types.NamespacedName]*apiv1.Service{
			{Namespace: "namespace1", Name: "service1"}: {
				TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace1", Name: "service1"},
				Spec: apiv1.ServiceSpec{
					Type: "ClusterIP",
					Ports: []apiv1.ServicePort{
						{Name: "http", Port: 80},
						{Name: "https", Port: 443},
					},
				},
			},
			{Namespace: "namespace2", Name: "service2"}: {
				TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace2", Name: "service2"},
				Spec: apiv1.ServiceSpec{
					Type: "ClusterIP",
					Ports: []apiv1.ServicePort{
						{Name: "http", Port: 80},
					},
				},
			},
			{Namespace: "namespace1", Name: "service3"}: {
				TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: "namespace1", Name: "service3"},
				Spec: apiv1.ServiceSpec{
					Type: "ClusterIP",
					Ports: []apiv1.ServicePort{
						{Name: "http", Port: 9200},
						{Name: "transport", Port: 9300},
					},
				},
			},
		}
		expected := map[types.NamespacedName]map[string]int32{
			{Namespace: "namespace1", Name: "service1"}: {"http": 80, "https": 443},
			{Namespace: "namespace2", Name: "service2"}: {"http": 80},
			{Namespace: "namespace1", Name: "service3"}: {"http": 9200, "transport": 9300},
		}

		require.Equal(t, expected, GroupServicePortsByPortName(services))
	})
}

func TestParseGRPCServiceMethod(t *testing.T) {
	testCases := []struct {
		name            string
		path            string
		expectedService string
		expectedMethod  string
	}{
		{
			name:            "empty path",
			path:            "",
			expectedService: "",
			expectedMethod:  "",
		},
		{
			name:            "root path",
			path:            "/",
			expectedService: "",
			expectedMethod:  "",
		},
		{
			name:            "service only",
			path:            "/UserService",
			expectedService: "UserService",
			expectedMethod:  "",
		},
		{
			name:            "service and method",
			path:            "/UserService/GetUser",
			expectedService: "UserService",
			expectedMethod:  "GetUser",
		},
		{
			name:            "service and method with extra path",
			path:            "/UserService/GetUser/extra",
			expectedService: "UserService",
			expectedMethod:  "GetUser/extra",
		},
		{
			name:            "path without leading slash",
			path:            "UserService/GetUser",
			expectedService: "UserService",
			expectedMethod:  "GetUser",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			service, method := ParseGRPCServiceMethod(tc.path)
			require.Equal(t, tc.expectedService, service)
			require.Equal(t, tc.expectedMethod, method)
		})
	}
}

func TestConvertHTTPFiltersToGRPCFilters(t *testing.T) {
	testCases := []struct {
		name                string
		httpFilters         []gwapiv1.HTTPRouteFilter
		expectedGRPCFilters []gwapiv1.GRPCRouteFilter
		expectedUnsupported []gwapiv1.HTTPRouteFilterType
	}{
		{
			name:                "empty filters",
			httpFilters:         []gwapiv1.HTTPRouteFilter{},
			expectedGRPCFilters: []gwapiv1.GRPCRouteFilter{},
			expectedUnsupported: []gwapiv1.HTTPRouteFilterType{},
		},
		{
			name: "request header modifier",
			httpFilters: []gwapiv1.HTTPRouteFilter{
				{
					Type: gwapiv1.HTTPRouteFilterRequestHeaderModifier,
					RequestHeaderModifier: &gwapiv1.HTTPHeaderFilter{
						Set: []gwapiv1.HTTPHeader{
							{Name: "X-Custom-Header", Value: "test"},
						},
						Add: []gwapiv1.HTTPHeader{
							{Name: "X-Additional", Value: "header"},
						},
						Remove: []string{"X-Remove"},
					},
				},
			},
			expectedGRPCFilters: []gwapiv1.GRPCRouteFilter{
				{
					Type: gwapiv1.GRPCRouteFilterRequestHeaderModifier,
					RequestHeaderModifier: &gwapiv1.HTTPHeaderFilter{
						Set: []gwapiv1.HTTPHeader{
							{Name: "X-Custom-Header", Value: "test"},
						},
						Add: []gwapiv1.HTTPHeader{
							{Name: "X-Additional", Value: "header"},
						},
						Remove: []string{"X-Remove"},
					},
				},
			},
			expectedUnsupported: []gwapiv1.HTTPRouteFilterType{},
		},
		{
			name: "response header modifier",
			httpFilters: []gwapiv1.HTTPRouteFilter{
				{
					Type: gwapiv1.HTTPRouteFilterResponseHeaderModifier,
					ResponseHeaderModifier: &gwapiv1.HTTPHeaderFilter{
						Set: []gwapiv1.HTTPHeader{
							{Name: "X-Response-Header", Value: "value"},
						},
					},
				},
			},
			expectedGRPCFilters: []gwapiv1.GRPCRouteFilter{
				{
					Type: gwapiv1.GRPCRouteFilterResponseHeaderModifier,
					ResponseHeaderModifier: &gwapiv1.HTTPHeaderFilter{
						Set: []gwapiv1.HTTPHeader{
							{Name: "X-Response-Header", Value: "value"},
						},
					},
				},
			},
			expectedUnsupported: []gwapiv1.HTTPRouteFilterType{},
		},
		{
			name: "unsupported filters are tracked",
			httpFilters: []gwapiv1.HTTPRouteFilter{
				{
					Type: gwapiv1.HTTPRouteFilterRequestRedirect,
					RequestRedirect: &gwapiv1.HTTPRequestRedirectFilter{
						Hostname: ptr.To(gwapiv1.PreciseHostname("example.com")),
					},
				},
				{
					Type: gwapiv1.HTTPRouteFilterRequestHeaderModifier,
					RequestHeaderModifier: &gwapiv1.HTTPHeaderFilter{
						Set: []gwapiv1.HTTPHeader{
							{Name: "X-Custom", Value: "test"},
						},
					},
				},
			},
			expectedGRPCFilters: []gwapiv1.GRPCRouteFilter{
				{
					Type: gwapiv1.GRPCRouteFilterRequestHeaderModifier,
					RequestHeaderModifier: &gwapiv1.HTTPHeaderFilter{
						Set: []gwapiv1.HTTPHeader{
							{Name: "X-Custom", Value: "test"},
						},
					},
				},
			},
			expectedUnsupported: []gwapiv1.HTTPRouteFilterType{
				gwapiv1.HTTPRouteFilterRequestRedirect,
			},
		},
		{
			name: "multiple unsupported filters",
			httpFilters: []gwapiv1.HTTPRouteFilter{
				{
					Type: gwapiv1.HTTPRouteFilterRequestRedirect,
				},
				{
					Type: gwapiv1.HTTPRouteFilterURLRewrite,
				},
				{
					Type: gwapiv1.HTTPRouteFilterRequestMirror,
				},
			},
			expectedGRPCFilters: []gwapiv1.GRPCRouteFilter{},
			expectedUnsupported: []gwapiv1.HTTPRouteFilterType{
				gwapiv1.HTTPRouteFilterRequestRedirect,
				gwapiv1.HTTPRouteFilterURLRewrite,
				gwapiv1.HTTPRouteFilterRequestMirror,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := ConvertHTTPFiltersToGRPCFilters(tc.httpFilters)
			require.Equal(t, tc.expectedGRPCFilters, result.GRPCFilters)
			require.Equal(t, tc.expectedUnsupported, result.UnsupportedTypes)
		})
	}
}

func TestRemoveGRPCRulesFromHTTPRoute(t *testing.T) {
	testCases := []struct {
		name           string
		httpRoute      *gwapiv1.HTTPRoute
		grpcServiceSet map[string]struct{}
		expectedRules  int
	}{
		{
			name: "no gRPC services - all rules remain",
			httpRoute: &gwapiv1.HTTPRoute{
				Spec: gwapiv1.HTTPRouteSpec{
					Rules: []gwapiv1.HTTPRouteRule{
						{
							BackendRefs: []gwapiv1.HTTPBackendRef{
								{
									BackendRef: gwapiv1.BackendRef{
										BackendObjectReference: gwapiv1.BackendObjectReference{
											Name: "http-service",
										},
									},
								},
							},
						},
					},
				},
			},
			grpcServiceSet: map[string]struct{}{},
			expectedRules:  1,
		},
		{
			name: "remove gRPC service rules",
			httpRoute: &gwapiv1.HTTPRoute{
				Spec: gwapiv1.HTTPRouteSpec{
					Rules: []gwapiv1.HTTPRouteRule{
						{
							BackendRefs: []gwapiv1.HTTPBackendRef{
								{
									BackendRef: gwapiv1.BackendRef{
										BackendObjectReference: gwapiv1.BackendObjectReference{
											Name: "grpc-service",
										},
									},
								},
							},
						},
						{
							BackendRefs: []gwapiv1.HTTPBackendRef{
								{
									BackendRef: gwapiv1.BackendRef{
										BackendObjectReference: gwapiv1.BackendObjectReference{
											Name: "http-service",
										},
									},
								},
							},
						},
					},
				},
			},
			grpcServiceSet: map[string]struct{}{"grpc-service": {}},
			expectedRules:  1,
		},
		{
			name: "mixed backend refs in same rule",
			httpRoute: &gwapiv1.HTTPRoute{
				Spec: gwapiv1.HTTPRouteSpec{
					Rules: []gwapiv1.HTTPRouteRule{
						{
							BackendRefs: []gwapiv1.HTTPBackendRef{
								{
									BackendRef: gwapiv1.BackendRef{
										BackendObjectReference: gwapiv1.BackendObjectReference{
											Name: "grpc-service",
										},
									},
								},
								{
									BackendRef: gwapiv1.BackendRef{
										BackendObjectReference: gwapiv1.BackendObjectReference{
											Name: "http-service",
										},
									},
								},
							},
						},
					},
				},
			},
			grpcServiceSet: map[string]struct{}{"grpc-service": {}},
			expectedRules:  1, // Rule remains but with only HTTP backend refs
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result := RemoveGRPCRulesFromHTTPRoute(tc.httpRoute, tc.grpcServiceSet)
			require.Len(t, result, tc.expectedRules)

			// For the mixed backend refs test, verify that only non-gRPC services remain
			if tc.name == "mixed backend refs in same rule" && len(result) > 0 {
				require.Len(t, result[0].BackendRefs, 1)
				require.Equal(t, gwapiv1.ObjectName("http-service"), result[0].BackendRefs[0].BackendRef.BackendObjectReference.Name)
			}
		})
	}
}

func TestCreateBackendTLSPolicy(t *testing.T) {
	testCases := []struct {
		name        string
		namespace   string
		policyName  string
		serviceName string
	}{
		{
			name:        "basic policy creation",
			namespace:   "default",
			policyName:  "test-ingress-ssl-service-backend-tls",
			serviceName: "ssl-service",
		},
		{
			name:        "different namespace",
			namespace:   "production",
			policyName:  "api-ingress-secure-api-backend-tls",
			serviceName: "secure-api",
		},
		{
			name:        "custom policy name",
			namespace:   "custom",
			policyName:  "my-custom-policy",
			serviceName: "custom-service",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			policy := CreateBackendTLSPolicy(tc.namespace, tc.policyName, tc.serviceName)

			require.Equal(t, tc.policyName, policy.Name)
			require.Equal(t, tc.namespace, policy.Namespace)
			require.Equal(t, "BackendTLSPolicy", policy.Kind)
			require.Equal(t, gwapiv1.GroupVersion.String(), policy.APIVersion)

			require.Len(t, policy.Spec.TargetRefs, 1)
			require.Equal(t, gwapiv1.ObjectName(tc.serviceName), policy.Spec.TargetRefs[0].Name)
			require.Equal(t, "", string(policy.Spec.TargetRefs[0].Group)) // Core group
			require.Equal(t, "Service", string(policy.Spec.TargetRefs[0].Kind))
		})
	}
}
