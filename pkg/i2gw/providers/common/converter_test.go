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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	providerir "github.com/kkk777-7/ingress2eg/pkg/i2gw/provider_intermediate"
	apiv1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func Test_ToIR(t *testing.T) {
	iPrefix := networkingv1.PathTypePrefix
	iExact := networkingv1.PathTypeExact
	gPathPrefix := gwapiv1.PathMatchPathPrefix
	gExact := gwapiv1.PathMatchExact

	testCases := []struct {
		name           string
		ingresses      []networkingv1.Ingress
		servicePorts   map[types.NamespacedName]map[string]int32
		expectedIR     providerir.ProviderIR
		expectedErrors field.ErrorList
	}{
		{
			name:           "empty",
			ingresses:      []networkingv1.Ingress{},
			servicePorts:   map[types.NamespacedName]map[string]int32{},
			expectedIR:     providerir.ProviderIR{},
			expectedErrors: field.ErrorList{},
		},
		{
			name: "simple ingress",
			ingresses: []networkingv1.Ingress{{
				ObjectMeta: metav1.ObjectMeta{Name: "simple", Namespace: "test"},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{{
						Host: "example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{{
									Path:     "/foo",
									PathType: &iPrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "example",
											Port: networkingv1.ServiceBackendPort{
												Number: 3000,
											},
										},
									},
								}},
							},
						},
					}},
					IngressClassName: ptr.To("simple"),
				},
			}},
			servicePorts: map[types.NamespacedName]map[string]int32{},
			expectedIR: providerir.ProviderIR{
				Gateways: map[types.NamespacedName]providerir.GatewayContext{
					{Namespace: "test", Name: "simple"}: {
						Gateway: gwapiv1.Gateway{
							ObjectMeta: metav1.ObjectMeta{Name: "simple", Namespace: "test"},
							Spec: gwapiv1.GatewaySpec{
								GatewayClassName: "simple",
								Listeners: []gwapiv1.Listener{{
									Name:     "example-com-http",
									Port:     80,
									Protocol: gwapiv1.HTTPProtocolType,
									Hostname: ptr.To(gwapiv1.Hostname("example.com")),
								}},
							},
						},
					},
				},
				HTTPRoutes: map[types.NamespacedName]providerir.HTTPRouteContext{
					{Namespace: "test", Name: "simple-example-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "simple-example-com", Namespace: "test"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "simple",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"example.com"},
								Rules: []gwapiv1.HTTPRouteRule{{
									Name: ptr.To(gwapiv1.SectionName("rule-prefix-foo")),
									Matches: []gwapiv1.HTTPRouteMatch{{
										Path: &gwapiv1.HTTPPathMatch{
											Type:  &gPathPrefix,
											Value: ptr.To("/foo"),
										},
									}},
									BackendRefs: []gwapiv1.HTTPBackendRef{{
										BackendRef: gwapiv1.BackendRef{
											BackendObjectReference: gwapiv1.BackendObjectReference{
												Name: "example",
												Port: ptr.To(gwapiv1.PortNumber(3000)),
											},
										},
									}},
								}},
							},
						},
					},
				},
			},
			expectedErrors: field.ErrorList{},
		},
		{
			name: "ingress with TLS",
			ingresses: []networkingv1.Ingress{{
				ObjectMeta: metav1.ObjectMeta{Name: "with-tls", Namespace: "test"},
				Spec: networkingv1.IngressSpec{
					TLS: []networkingv1.IngressTLS{{
						Hosts:      []string{"example.com"},
						SecretName: "example-cert",
					}},
					Rules: []networkingv1.IngressRule{{
						Host: "example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{{
									Path:     "/foo",
									PathType: &iPrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "example",
											Port: networkingv1.ServiceBackendPort{
												Number: 3000,
											},
										},
									},
								}},
							},
						},
					}},
					IngressClassName: ptr.To("with-tls"),
				},
			}},
			servicePorts: map[types.NamespacedName]map[string]int32{},
			expectedIR: providerir.ProviderIR{
				Gateways: map[types.NamespacedName]providerir.GatewayContext{
					{Namespace: "test", Name: "with-tls"}: {
						Gateway: gwapiv1.Gateway{
							ObjectMeta: metav1.ObjectMeta{Name: "with-tls", Namespace: "test"},
							Spec: gwapiv1.GatewaySpec{
								GatewayClassName: "with-tls",
								Listeners: []gwapiv1.Listener{{
									Name:     "example-com-http",
									Port:     80,
									Protocol: gwapiv1.HTTPProtocolType,
									Hostname: ptr.To(gwapiv1.Hostname("example.com")),
								}, {
									Name:     "example-com-https",
									Port:     443,
									Protocol: gwapiv1.HTTPSProtocolType,
									Hostname: ptr.To(gwapiv1.Hostname("example.com")),
									TLS: &gwapiv1.ListenerTLSConfig{
										CertificateRefs: []gwapiv1.SecretObjectReference{{
											Name: "example-cert",
										}},
									},
								}},
							},
						},
					},
				},
				HTTPRoutes: map[types.NamespacedName]providerir.HTTPRouteContext{
					{Namespace: "test", Name: "with-tls-example-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "with-tls-example-com", Namespace: "test"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "with-tls",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"example.com"},
								Rules: []gwapiv1.HTTPRouteRule{{
									Name: ptr.To(gwapiv1.SectionName("rule-prefix-foo")),
									Matches: []gwapiv1.HTTPRouteMatch{{
										Path: &gwapiv1.HTTPPathMatch{
											Type:  &gPathPrefix,
											Value: ptr.To("/foo"),
										},
									}},
									BackendRefs: []gwapiv1.HTTPBackendRef{{
										BackendRef: gwapiv1.BackendRef{
											BackendObjectReference: gwapiv1.BackendObjectReference{
												Name: "example",
												Port: ptr.To(gwapiv1.PortNumber(3000)),
											},
										},
									}},
								}},
							},
						},
					},
				},
			},
			expectedErrors: field.ErrorList{},
		},
		{
			name: "ingress with custom and default backend",
			ingresses: []networkingv1.Ingress{{
				ObjectMeta: metav1.ObjectMeta{Name: "net", Namespace: "different"},
				Spec: networkingv1.IngressSpec{
					IngressClassName: ptr.To("example-proxy"),
					Rules: []networkingv1.IngressRule{{
						Host: "example.net",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{{
									Path:     "/bar",
									PathType: &iExact,
									Backend: networkingv1.IngressBackend{
										Resource: &apiv1.TypedLocalObjectReference{
											Name:     "custom",
											Kind:     "StorageBucket",
											APIGroup: ptr.To("vendor.example.com"),
										},
									},
								}},
							},
						},
					}},
					DefaultBackend: &networkingv1.IngressBackend{
						Service: &networkingv1.IngressServiceBackend{
							Name: "default",
							Port: networkingv1.ServiceBackendPort{
								Number: 8080,
							},
						},
					},
				},
			}},
			servicePorts: map[types.NamespacedName]map[string]int32{},
			expectedIR: providerir.ProviderIR{
				Gateways: map[types.NamespacedName]providerir.GatewayContext{
					{Namespace: "different", Name: "example-proxy"}: {
						Gateway: gwapiv1.Gateway{
							ObjectMeta: metav1.ObjectMeta{Name: "example-proxy", Namespace: "different"},
							Spec: gwapiv1.GatewaySpec{
								GatewayClassName: "example-proxy",
								Listeners: []gwapiv1.Listener{{
									Name:     "example-net-http",
									Port:     80,
									Protocol: gwapiv1.HTTPProtocolType,
									Hostname: ptr.To(gwapiv1.Hostname("example.net")),
								}},
							},
						},
					},
				},
				HTTPRoutes: map[types.NamespacedName]providerir.HTTPRouteContext{
					{Namespace: "different", Name: "net-example-net"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "net-example-net", Namespace: "different"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "example-proxy",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"example.net"},
								Rules: []gwapiv1.HTTPRouteRule{{
									Name: ptr.To(gwapiv1.SectionName("rule-exact-bar")),
									Matches: []gwapiv1.HTTPRouteMatch{{
										Path: &gwapiv1.HTTPPathMatch{
											Type:  &gExact,
											Value: ptr.To("/bar"),
										},
									}},
									BackendRefs: []gwapiv1.HTTPBackendRef{{
										BackendRef: gwapiv1.BackendRef{
											BackendObjectReference: gwapiv1.BackendObjectReference{
												Name:  "custom",
												Group: ptr.To(gwapiv1.Group("vendor.example.com")),
												Kind:  ptr.To(gwapiv1.Kind("StorageBucket")),
											},
										},
									}},
								}},
							},
						},
					},
					{Namespace: "different", Name: "net-default-backend"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "net-default-backend", Namespace: "different"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "example-proxy",
									}},
								},
								Rules: []gwapiv1.HTTPRouteRule{{
									BackendRefs: []gwapiv1.HTTPBackendRef{{
										BackendRef: gwapiv1.BackendRef{
											BackendObjectReference: gwapiv1.BackendObjectReference{
												Name: "default",
												Port: ptr.To(gwapiv1.PortNumber(8080)),
											},
										}},
									}},
								},
							},
						},
					},
				},
			},
			expectedErrors: field.ErrorList{},
		},
		{
			name: "named ports",
			ingresses: []networkingv1.Ingress{{
				ObjectMeta: metav1.ObjectMeta{Name: "named-ports", Namespace: "test"},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{{
						Host: "example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{{
									Path:     "/foo",
									PathType: &iPrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "example",
											Port: networkingv1.ServiceBackendPort{
												Name: "http",
											},
										},
									},
								}},
							},
						},
					}},
					IngressClassName: ptr.To("named-ports"),
				},
			}},
			servicePorts: map[types.NamespacedName]map[string]int32{
				{Namespace: "test", Name: "example"}:  {"http": 3000},
				{Namespace: "test", Name: "example2"}: {"http": 8080},
			},
			expectedIR: providerir.ProviderIR{
				Gateways: map[types.NamespacedName]providerir.GatewayContext{
					{Namespace: "test", Name: "named-ports"}: {
						Gateway: gwapiv1.Gateway{
							ObjectMeta: metav1.ObjectMeta{Name: "named-ports", Namespace: "test"},
							Spec: gwapiv1.GatewaySpec{
								GatewayClassName: "named-ports",
								Listeners: []gwapiv1.Listener{{
									Name:     "example-com-http",
									Port:     80,
									Protocol: gwapiv1.HTTPProtocolType,
									Hostname: ptr.To(gwapiv1.Hostname("example.com")),
								}},
							},
						},
					},
				},
				HTTPRoutes: map[types.NamespacedName]providerir.HTTPRouteContext{
					{Namespace: "test", Name: "named-ports-example-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "named-ports-example-com", Namespace: "test"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "named-ports",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"example.com"},
								Rules: []gwapiv1.HTTPRouteRule{{
									Name: ptr.To(gwapiv1.SectionName("rule-prefix-foo")),
									Matches: []gwapiv1.HTTPRouteMatch{{
										Path: &gwapiv1.HTTPPathMatch{
											Type:  &gPathPrefix,
											Value: ptr.To("/foo"),
										},
									}},
									BackendRefs: []gwapiv1.HTTPBackendRef{{
										BackendRef: gwapiv1.BackendRef{
											BackendObjectReference: gwapiv1.BackendObjectReference{
												Name: "example",
												Port: ptr.To(gwapiv1.PortNumber(3000)),
											},
										},
									}},
								}},
							},
						},
					},
				},
			},
			expectedErrors: field.ErrorList{},
		},
		{
			name: "missing named ports",
			ingresses: []networkingv1.Ingress{{
				ObjectMeta: metav1.ObjectMeta{Name: "named-ports", Namespace: "test"},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{{
						Host: "example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{{
									Path:     "/foo",
									PathType: &iPrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "example",
											Port: networkingv1.ServiceBackendPort{
												Name: "http",
											},
										},
									},
								}},
							},
						},
					}},
					IngressClassName: ptr.To("named-ports"),
				},
			}},
			servicePorts: map[types.NamespacedName]map[string]int32{
				{Namespace: "test", Name: "example2"}: {"http": 8080},
			},
			expectedIR: providerir.ProviderIR{
				Gateways:   map[types.NamespacedName]providerir.GatewayContext{},
				HTTPRoutes: map[types.NamespacedName]providerir.HTTPRouteContext{},
			},
			expectedErrors: field.ErrorList{field.Invalid(field.NewPath(""), "", "")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			ir, errs := ToIR(tc.ingresses, tc.servicePorts, i2gw.ProviderImplementationSpecificOptions{})

			if len(ir.HTTPRoutes) != len(tc.expectedIR.HTTPRoutes) {
				t.Errorf("Expected %d HTTPRoutes, got %d: %+v",
					len(tc.expectedIR.HTTPRoutes), len(ir.HTTPRoutes), ir.HTTPRoutes)
			} else {
				for i, gotHTTPRouteContext := range ir.HTTPRoutes {
					key := types.NamespacedName{Namespace: gotHTTPRouteContext.HTTPRoute.Namespace, Name: gotHTTPRouteContext.HTTPRoute.Name}
					wantHTTPRouteContext := tc.expectedIR.HTTPRoutes[key]
					wantHTTPRouteContext.HTTPRoute.SetGroupVersionKind(HTTPRouteGVK)
					if !apiequality.Semantic.DeepEqual(gotHTTPRouteContext.HTTPRoute, wantHTTPRouteContext.HTTPRoute) {
						t.Errorf("Expected HTTPRoute %s to be %+v\n Got: %+v\n Diff: %s", i, wantHTTPRouteContext.HTTPRoute, gotHTTPRouteContext.HTTPRoute, cmp.Diff(wantHTTPRouteContext.HTTPRoute, gotHTTPRouteContext.HTTPRoute))
					}
				}
			}

			if len(ir.Gateways) != len(tc.expectedIR.Gateways) {
				t.Errorf("Expected %d Gateways, got %d: %+v",
					len(tc.expectedIR.Gateways), len(ir.Gateways), ir.Gateways)
			} else {
				for i, gotGatewayContext := range ir.Gateways {
					key := types.NamespacedName{Namespace: gotGatewayContext.Gateway.Namespace, Name: gotGatewayContext.Gateway.Name}
					wantGatewayContext := tc.expectedIR.Gateways[key]
					wantGatewayContext.Gateway.SetGroupVersionKind(GatewayGVK)
					if !apiequality.Semantic.DeepEqual(gotGatewayContext.Gateway, wantGatewayContext.Gateway) {
						t.Errorf("Expected Gateway %s to be %+v\n Got: %+v\n Diff: %s", i, wantGatewayContext.Gateway, gotGatewayContext.Gateway, cmp.Diff(wantGatewayContext.Gateway, gotGatewayContext.Gateway))
					}
				}
			}

			if len(errs) != len(tc.expectedErrors) {
				t.Errorf("Expected %d errors, got %d: %+v", len(tc.expectedErrors), len(errs), errs)
			} else {
				for i, e := range errs {
					if errors.Is(e, tc.expectedErrors[i]) {
						t.Errorf("Unexpected error message at %d index. Got %s, want: %s", i, e, tc.expectedErrors[i])
					}
				}
			}
		})
	}
}
