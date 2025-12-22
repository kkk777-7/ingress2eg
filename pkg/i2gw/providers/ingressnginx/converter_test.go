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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	providerir "github.com/kkk777-7/ingress2eg/pkg/i2gw/provider_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/providers/common"
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
	//iExact := networkingv1.PathTypeExact
	isPathType := networkingv1.PathTypeImplementationSpecific
	gPathPrefix := gwapiv1.PathMatchPathPrefix
	//gExact := gwapiv1.PathMatchExact

	testCases := []struct {
		name           string
		ingresses      OrderedIngressMap
		expectedIR     providerir.ProviderIR
		expectedErrors field.ErrorList
	}{
		{
			name: "canary deployment",
			ingresses: OrderedIngressMap{
				ingressNames: []types.NamespacedName{{Namespace: "default", Name: "production"}, {Namespace: "default", Name: "canary"}},
				ingressObjects: map[types.NamespacedName]*networkingv1.Ingress{
					{Namespace: "default", Name: "production"}: {
						ObjectMeta: metav1.ObjectMeta{Name: "production", Namespace: "default"},
						Spec: networkingv1.IngressSpec{
							IngressClassName: ptr.To("ingress-nginx"),
							Rules: []networkingv1.IngressRule{{
								Host: "echo.prod.mydomain.com",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{{
											Path:     "/",
											PathType: &iPrefix,
											Backend: networkingv1.IngressBackend{
												Resource: &apiv1.TypedLocalObjectReference{
													Name:     "production",
													Kind:     "StorageBucket",
													APIGroup: ptr.To("vendor.example.com"),
												},
											},
										}},
									},
								},
							}},
						},
					},
					{Namespace: "default", Name: "canary"}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "canary",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.ingress.kubernetes.io/canary":        "true",
								"nginx.ingress.kubernetes.io/canary-weight": "20",
							},
						},
						Spec: networkingv1.IngressSpec{
							IngressClassName: ptr.To("ingress-nginx"),
							Rules: []networkingv1.IngressRule{{
								Host: "echo.prod.mydomain.com",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{{
											Path:     "/",
											PathType: &iPrefix,
											Backend: networkingv1.IngressBackend{
												Resource: &apiv1.TypedLocalObjectReference{
													Name:     "canary",
													Kind:     "StorageBucket",
													APIGroup: ptr.To("vendor.example.com"),
												},
											},
										}},
									},
								},
							}},
						},
					},
				},
			},
			expectedIR: providerir.ProviderIR{
				Gateways: map[types.NamespacedName]providerir.GatewayContext{
					{Namespace: "default", Name: "ingress-nginx"}: {
						Gateway: gwapiv1.Gateway{
							ObjectMeta: metav1.ObjectMeta{Name: "ingress-nginx", Namespace: "default"},
							Spec: gwapiv1.GatewaySpec{
								GatewayClassName: "ingress-nginx",
								Listeners: []gwapiv1.Listener{{
									Name:     "echo-prod-mydomain-com-http",
									Port:     80,
									Protocol: gwapiv1.HTTPProtocolType,
									Hostname: ptr.To(gwapiv1.Hostname("echo.prod.mydomain.com")),
								}},
							},
						},
					},
				},
				HTTPRoutes: map[types.NamespacedName]providerir.HTTPRouteContext{
					{Namespace: "default", Name: "production-echo-prod-mydomain-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "production-echo-prod-mydomain-com", Namespace: "default"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "ingress-nginx",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"echo.prod.mydomain.com"},
								Rules: []gwapiv1.HTTPRouteRule{{
									Matches: []gwapiv1.HTTPRouteMatch{{
										Path: &gwapiv1.HTTPPathMatch{
											Type:  &gPathPrefix,
											Value: ptr.To("/"),
										},
									}},
									BackendRefs: []gwapiv1.HTTPBackendRef{
										{
											BackendRef: gwapiv1.BackendRef{
												BackendObjectReference: gwapiv1.BackendObjectReference{
													Name:  "production",
													Group: ptr.To(gwapiv1.Group("vendor.example.com")),
													Kind:  ptr.To(gwapiv1.Kind("StorageBucket")),
												},
												Weight: ptr.To(int32(80)),
											},
										},
										{
											BackendRef: gwapiv1.BackendRef{
												BackendObjectReference: gwapiv1.BackendObjectReference{
													Name:  "canary",
													Group: ptr.To(gwapiv1.Group("vendor.example.com")),
													Kind:  ptr.To(gwapiv1.Kind("StorageBucket")),
												},
												Weight: ptr.To(int32(20)),
											},
										},
									},
								}},
							},
						},
					},
				},
			},
			expectedErrors: field.ErrorList{},
		},
		{
			name: "canary deployment total weight",
			ingresses: OrderedIngressMap{
				ingressNames: []types.NamespacedName{{Namespace: "default", Name: "production"}, {Namespace: "default", Name: "canary"}},
				ingressObjects: map[types.NamespacedName]*networkingv1.Ingress{
					{Namespace: "default", Name: "production"}: {
						ObjectMeta: metav1.ObjectMeta{Name: "production", Namespace: "default"},
						Spec: networkingv1.IngressSpec{
							IngressClassName: ptr.To("ingress-nginx"),
							Rules: []networkingv1.IngressRule{{
								Host: "echo.prod.mydomain.com",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{{
											Path:     "/",
											PathType: &iPrefix,
											Backend: networkingv1.IngressBackend{
												Resource: &apiv1.TypedLocalObjectReference{
													Name:     "production",
													Kind:     "StorageBucket",
													APIGroup: ptr.To("vendor.example.com"),
												},
											},
										}},
									},
								},
							}},
						},
					},
					{Namespace: "default", Name: "canary"}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "canary",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.ingress.kubernetes.io/canary":              "true",
								"nginx.ingress.kubernetes.io/canary-weight":       "20",
								"nginx.ingress.kubernetes.io/canary-weight-total": "200",
							},
						},
						Spec: networkingv1.IngressSpec{
							IngressClassName: ptr.To("ingress-nginx"),
							Rules: []networkingv1.IngressRule{{
								Host: "echo.prod.mydomain.com",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{{
											Path:     "/",
											PathType: &iPrefix,
											Backend: networkingv1.IngressBackend{
												Resource: &apiv1.TypedLocalObjectReference{
													Name:     "canary",
													Kind:     "StorageBucket",
													APIGroup: ptr.To("vendor.example.com"),
												},
											},
										}},
									},
								},
							}},
						},
					},
				},
			},
			expectedIR: providerir.ProviderIR{
				Gateways: map[types.NamespacedName]providerir.GatewayContext{
					{Namespace: "default", Name: "ingress-nginx"}: {
						Gateway: gwapiv1.Gateway{
							ObjectMeta: metav1.ObjectMeta{Name: "ingress-nginx", Namespace: "default"},
							Spec: gwapiv1.GatewaySpec{
								GatewayClassName: "ingress-nginx",
								Listeners: []gwapiv1.Listener{{
									Name:     "echo-prod-mydomain-com-http",
									Port:     80,
									Protocol: gwapiv1.HTTPProtocolType,
									Hostname: ptr.To(gwapiv1.Hostname("echo.prod.mydomain.com")),
								}},
							},
						},
					},
				},
				HTTPRoutes: map[types.NamespacedName]providerir.HTTPRouteContext{
					{Namespace: "default", Name: "production-echo-prod-mydomain-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "production-echo-prod-mydomain-com", Namespace: "default"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "ingress-nginx",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"echo.prod.mydomain.com"},
								Rules: []gwapiv1.HTTPRouteRule{{
									Matches: []gwapiv1.HTTPRouteMatch{{
										Path: &gwapiv1.HTTPPathMatch{
											Type:  &gPathPrefix,
											Value: ptr.To("/"),
										},
									}},
									BackendRefs: []gwapiv1.HTTPBackendRef{
										{
											BackendRef: gwapiv1.BackendRef{
												BackendObjectReference: gwapiv1.BackendObjectReference{
													Name:  "production",
													Group: ptr.To(gwapiv1.Group("vendor.example.com")),
													Kind:  ptr.To(gwapiv1.Kind("StorageBucket")),
												},
												Weight: ptr.To(int32(180)),
											},
										},
										{
											BackendRef: gwapiv1.BackendRef{
												BackendObjectReference: gwapiv1.BackendObjectReference{
													Name:  "canary",
													Group: ptr.To(gwapiv1.Group("vendor.example.com")),
													Kind:  ptr.To(gwapiv1.Kind("StorageBucket")),
												},
												Weight: ptr.To(int32(20)),
											},
										},
									},
								}},
							},
						},
					},
				},
			},
			expectedErrors: field.ErrorList{},
		},
		{
			name: "ImplementationSpecific HTTPRouteMatching",
			ingresses: OrderedIngressMap{
				ingressNames: []types.NamespacedName{{Namespace: "default", Name: "implementation-specific-regex"}},
				ingressObjects: map[types.NamespacedName]*networkingv1.Ingress{
					{Namespace: "default", Name: "implementation-specific-regex"}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "implementation-specific-regex",
							Namespace: "default",
						},
						Spec: networkingv1.IngressSpec{
							IngressClassName: ptr.To("ingress-nginx"),
							Rules: []networkingv1.IngressRule{{
								Host: "test.mydomain.com",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{{
											Path:     "/~/echo/**/test",
											PathType: &isPathType,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "test",
													Port: networkingv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										}},
									},
								},
							}},
						},
					},
				},
			},
			expectedIR: providerir.ProviderIR{},
			expectedErrors: field.ErrorList{
				{
					Type:     field.ErrorTypeInvalid,
					Field:    "spec.rules[0].http.paths[0].pathType",
					BadValue: ptr.To("ImplementationSpecific"),
					Detail:   "implementationSpecific path type is not supported in generic translation, and your provider does not provide custom support to translate it",
				},
			},
		},
		{
			name: "multiple rules with TLS",
			ingresses: OrderedIngressMap{
				ingressNames: []types.NamespacedName{{Namespace: "default", Name: "example-ingress"}},
				ingressObjects: map[types.NamespacedName]*networkingv1.Ingress{
					{Namespace: "default", Name: "example-ingress"}: {
						ObjectMeta: metav1.ObjectMeta{Name: "example-ingress", Namespace: "default"},
						Spec: networkingv1.IngressSpec{
							IngressClassName: ptr.To("nginx"),
							TLS: []networkingv1.IngressTLS{{
								Hosts: []string{
									"foo.example.com",
									"bar.example.com",
								},
								SecretName: "example-com",
							}},
							Rules: []networkingv1.IngressRule{
								{
									Host: "foo.example.com",
									IngressRuleValue: networkingv1.IngressRuleValue{
										HTTP: &networkingv1.HTTPIngressRuleValue{
											Paths: []networkingv1.HTTPIngressPath{
												{
													Path:     "/",
													PathType: &iPrefix,
													Backend: networkingv1.IngressBackend{
														Service: &networkingv1.IngressServiceBackend{
															Name: "foo-app",
															Port: networkingv1.ServiceBackendPort{Number: 80},
														},
													},
												},
												{
													Path:     "/orders",
													PathType: &iPrefix,
													Backend: networkingv1.IngressBackend{
														Service: &networkingv1.IngressServiceBackend{
															Name: "foo-orders-app",
															Port: networkingv1.ServiceBackendPort{Number: 80},
														},
													},
												},
											},
										},
									},
								},
								{
									Host: "bar.example.com",
									IngressRuleValue: networkingv1.IngressRuleValue{
										HTTP: &networkingv1.HTTPIngressRuleValue{
											Paths: []networkingv1.HTTPIngressPath{
												{
													Path:     "/",
													PathType: &iPrefix,
													Backend: networkingv1.IngressBackend{
														Service: &networkingv1.IngressServiceBackend{
															Name: "bar-app",
															Port: networkingv1.ServiceBackendPort{Number: 80},
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
				},
			},
			expectedIR: providerir.ProviderIR{
				Gateways: map[types.NamespacedName]providerir.GatewayContext{
					{Namespace: "default", Name: "nginx"}: {
						Gateway: gwapiv1.Gateway{
							ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "default"},
							Spec: gwapiv1.GatewaySpec{
								GatewayClassName: "nginx",
								Listeners: []gwapiv1.Listener{
									{
										Name:     "bar-example-com-http",
										Port:     80,
										Protocol: gwapiv1.HTTPProtocolType,
										Hostname: ptr.To(gwapiv1.Hostname("bar.example.com")),
									},
									{
										Name:     "bar-example-com-https",
										Port:     443,
										Protocol: gwapiv1.HTTPSProtocolType,
										Hostname: ptr.To(gwapiv1.Hostname("bar.example.com")),
										TLS: &gwapiv1.ListenerTLSConfig{
											CertificateRefs: []gwapiv1.SecretObjectReference{
												{Name: "example-com"},
											},
										},
									},
									{
										Name:     "foo-example-com-http",
										Port:     80,
										Protocol: gwapiv1.HTTPProtocolType,
										Hostname: ptr.To(gwapiv1.Hostname("foo.example.com")),
									},
									{
										Name:     "foo-example-com-https",
										Port:     443,
										Protocol: gwapiv1.HTTPSProtocolType,
										Hostname: ptr.To(gwapiv1.Hostname("foo.example.com")),
										TLS: &gwapiv1.ListenerTLSConfig{
											CertificateRefs: []gwapiv1.SecretObjectReference{
												{Name: "example-com"},
											},
										},
									},
								},
							},
						},
					},
				},
				HTTPRoutes: map[types.NamespacedName]providerir.HTTPRouteContext{
					{Namespace: "default", Name: "example-ingress-bar-example-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "example-ingress-bar-example-com", Namespace: "default"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "nginx",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"bar.example.com"},
								Rules: []gwapiv1.HTTPRouteRule{{
									Matches: []gwapiv1.HTTPRouteMatch{{
										Path: &gwapiv1.HTTPPathMatch{
											Type:  &gPathPrefix,
											Value: ptr.To("/"),
										},
									}},
									BackendRefs: []gwapiv1.HTTPBackendRef{
										{
											BackendRef: gwapiv1.BackendRef{
												BackendObjectReference: gwapiv1.BackendObjectReference{
													Name: "bar-app",
													Port: ptr.To(gwapiv1.PortNumber(80)),
												},
											},
										},
									},
								}},
							},
						},
					},
					{Namespace: "default", Name: "example-ingress-foo-example-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "example-ingress-foo-example-com", Namespace: "default"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "nginx",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"foo.example.com"},
								Rules: []gwapiv1.HTTPRouteRule{
									{
										Matches: []gwapiv1.HTTPRouteMatch{{
											Path: &gwapiv1.HTTPPathMatch{
												Type:  &gPathPrefix,
												Value: ptr.To("/"),
											},
										}},
										BackendRefs: []gwapiv1.HTTPBackendRef{
											{
												BackendRef: gwapiv1.BackendRef{
													BackendObjectReference: gwapiv1.BackendObjectReference{
														Name: "foo-app",
														Port: ptr.To(gwapiv1.PortNumber(80)),
													},
												},
											},
										},
									},
									{
										Matches: []gwapiv1.HTTPRouteMatch{{
											Path: &gwapiv1.HTTPPathMatch{
												Type:  &gPathPrefix,
												Value: ptr.To("/orders"),
											},
										}},
										BackendRefs: []gwapiv1.HTTPBackendRef{
											{
												BackendRef: gwapiv1.BackendRef{
													BackendObjectReference: gwapiv1.BackendObjectReference{
														Name: "foo-orders-app",
														Port: ptr.To(gwapiv1.PortNumber(80)),
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
			},
			expectedErrors: field.ErrorList{},
		},
		{
			name: "multiple rules with canary",
			ingresses: OrderedIngressMap{
				ingressNames: []types.NamespacedName{
					{Namespace: "default", Name: "example-ingress"},
					{Namespace: "default", Name: "example-ingress-canary"},
				},
				ingressObjects: map[types.NamespacedName]*networkingv1.Ingress{
					{Namespace: "default", Name: "example-ingress"}: {
						ObjectMeta: metav1.ObjectMeta{Name: "example-ingress", Namespace: "default"},
						Spec: networkingv1.IngressSpec{
							IngressClassName: ptr.To("nginx"),
							Rules: []networkingv1.IngressRule{
								{
									Host: "foo.example.com",
									IngressRuleValue: networkingv1.IngressRuleValue{
										HTTP: &networkingv1.HTTPIngressRuleValue{
											Paths: []networkingv1.HTTPIngressPath{
												{
													Path:     "/",
													PathType: &iPrefix,
													Backend: networkingv1.IngressBackend{
														Service: &networkingv1.IngressServiceBackend{
															Name: "foo-app",
															Port: networkingv1.ServiceBackendPort{Number: 80},
														},
													},
												},
												{
													Path:     "/orders",
													PathType: &iPrefix,
													Backend: networkingv1.IngressBackend{
														Service: &networkingv1.IngressServiceBackend{
															Name: "foo-orders-app",
															Port: networkingv1.ServiceBackendPort{Number: 80},
														},
													},
												},
											},
										},
									},
								},
								{
									Host: "bar.example.com",
									IngressRuleValue: networkingv1.IngressRuleValue{
										HTTP: &networkingv1.HTTPIngressRuleValue{
											Paths: []networkingv1.HTTPIngressPath{
												{
													Path:     "/",
													PathType: &iPrefix,
													Backend: networkingv1.IngressBackend{
														Service: &networkingv1.IngressServiceBackend{
															Name: "bar-app",
															Port: networkingv1.ServiceBackendPort{Number: 80},
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
					{Namespace: "default", Name: "example-ingress-canary"}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "example-ingress-canary",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.ingress.kubernetes.io/canary":        "true",
								"nginx.ingress.kubernetes.io/canary-weight": "30",
							},
						},
						Spec: networkingv1.IngressSpec{
							IngressClassName: ptr.To("nginx"),
							Rules: []networkingv1.IngressRule{
								{
									Host: "bar.example.com",
									IngressRuleValue: networkingv1.IngressRuleValue{
										HTTP: &networkingv1.HTTPIngressRuleValue{
											Paths: []networkingv1.HTTPIngressPath{
												{
													Path:     "/",
													PathType: &iPrefix,
													Backend: networkingv1.IngressBackend{
														Service: &networkingv1.IngressServiceBackend{
															Name: "bar-app-canary",
															Port: networkingv1.ServiceBackendPort{Number: 80},
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
				},
			},
			expectedIR: providerir.ProviderIR{
				Gateways: map[types.NamespacedName]providerir.GatewayContext{
					{Namespace: "default", Name: "nginx"}: {
						Gateway: gwapiv1.Gateway{
							ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "default"},
							Spec: gwapiv1.GatewaySpec{
								GatewayClassName: "nginx",
								Listeners: []gwapiv1.Listener{
									{
										Name:     "bar-example-com-http",
										Port:     80,
										Protocol: gwapiv1.HTTPProtocolType,
										Hostname: ptr.To(gwapiv1.Hostname("bar.example.com")),
									},
									{
										Name:     "foo-example-com-http",
										Port:     80,
										Protocol: gwapiv1.HTTPProtocolType,
										Hostname: ptr.To(gwapiv1.Hostname("foo.example.com")),
									},
								},
							},
						},
					},
				},
				HTTPRoutes: map[types.NamespacedName]providerir.HTTPRouteContext{
					{Namespace: "default", Name: "example-ingress-bar-example-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "example-ingress-bar-example-com", Namespace: "default"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "nginx",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"bar.example.com"},
								Rules: []gwapiv1.HTTPRouteRule{{
									Matches: []gwapiv1.HTTPRouteMatch{{
										Path: &gwapiv1.HTTPPathMatch{
											Type:  &gPathPrefix,
											Value: ptr.To("/"),
										},
									}},
									BackendRefs: []gwapiv1.HTTPBackendRef{
										{
											BackendRef: gwapiv1.BackendRef{
												BackendObjectReference: gwapiv1.BackendObjectReference{
													Name: "bar-app",
													Port: ptr.To(gwapiv1.PortNumber(80)),
												},
												Weight: ptr.To[int32](70),
											},
										},
										{
											BackendRef: gwapiv1.BackendRef{
												BackendObjectReference: gwapiv1.BackendObjectReference{
													Name: "bar-app-canary",
													Port: ptr.To(gwapiv1.PortNumber(80)),
												},
												Weight: ptr.To[int32](30),
											},
										},
									},
								}},
							},
						},
					},
					{Namespace: "default", Name: "example-ingress-foo-example-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "example-ingress-foo-example-com", Namespace: "default"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "nginx",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"foo.example.com"},
								Rules: []gwapiv1.HTTPRouteRule{
									{
										Matches: []gwapiv1.HTTPRouteMatch{{
											Path: &gwapiv1.HTTPPathMatch{
												Type:  &gPathPrefix,
												Value: ptr.To("/"),
											},
										}},
										BackendRefs: []gwapiv1.HTTPBackendRef{
											{
												BackendRef: gwapiv1.BackendRef{
													BackendObjectReference: gwapiv1.BackendObjectReference{
														Name: "foo-app",
														Port: ptr.To(gwapiv1.PortNumber(80)),
													},
												},
											},
										},
									},
									{
										Matches: []gwapiv1.HTTPRouteMatch{{
											Path: &gwapiv1.HTTPPathMatch{
												Type:  &gPathPrefix,
												Value: ptr.To("/orders"),
											},
										}},
										BackendRefs: []gwapiv1.HTTPBackendRef{
											{
												BackendRef: gwapiv1.BackendRef{
													BackendObjectReference: gwapiv1.BackendObjectReference{
														Name: "foo-orders-app",
														Port: ptr.To(gwapiv1.PortNumber(80)),
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
			},
			expectedErrors: field.ErrorList{},
		},
		{
			name: "canary with same service in different rules - based on bad.yaml",
			ingresses: OrderedIngressMap{
				ingressNames: []types.NamespacedName{{Namespace: "default", Name: "production-ingress"}, {Namespace: "default", Name: "canary-ingress"}},
				ingressObjects: map[types.NamespacedName]*networkingv1.Ingress{
					{Namespace: "default", Name: "production-ingress"}: {
						ObjectMeta: metav1.ObjectMeta{Name: "production-ingress", Namespace: "default"},
						Spec: networkingv1.IngressSpec{
							IngressClassName: ptr.To("nginx"),
							Rules: []networkingv1.IngressRule{{
								Host: "api.example.com",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path:     "/api",
												PathType: &iPrefix,
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "api-service-v1",
														Port: networkingv1.ServiceBackendPort{Number: 80},
													},
												},
											},
											{
												Path:     "/admin",
												PathType: &iPrefix,
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "admin-service",
														Port: networkingv1.ServiceBackendPort{Number: 80},
													},
												},
											},
										},
									},
								},
							}},
						},
					},
					{Namespace: "default", Name: "canary-ingress"}: {
						ObjectMeta: metav1.ObjectMeta{
							Name:      "canary-ingress",
							Namespace: "default",
							Annotations: map[string]string{
								"nginx.ingress.kubernetes.io/canary":        "true",
								"nginx.ingress.kubernetes.io/canary-weight": "10",
							},
						},
						Spec: networkingv1.IngressSpec{
							IngressClassName: ptr.To("nginx"),
							Rules: []networkingv1.IngressRule{{
								Host: "api.example.com",
								IngressRuleValue: networkingv1.IngressRuleValue{
									HTTP: &networkingv1.HTTPIngressRuleValue{
										Paths: []networkingv1.HTTPIngressPath{
											{
												Path:     "/api",
												PathType: &iPrefix,
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "api-service-v2",
														Port: networkingv1.ServiceBackendPort{Number: 80},
													},
												},
											},
											{
												Path:     "/admin",
												PathType: &iPrefix,
												Backend: networkingv1.IngressBackend{
													Service: &networkingv1.IngressServiceBackend{
														Name: "api-service-v1", // Same service as production's "/api" path!
														Port: networkingv1.ServiceBackendPort{Number: 80},
													},
												},
											},
										},
									},
								},
							}},
						},
					},
				},
			},
			expectedIR: providerir.ProviderIR{
				Gateways: map[types.NamespacedName]providerir.GatewayContext{
					{Namespace: "default", Name: "nginx"}: {
						Gateway: gwapiv1.Gateway{
							ObjectMeta: metav1.ObjectMeta{Name: "nginx", Namespace: "default"},
							Spec: gwapiv1.GatewaySpec{
								GatewayClassName: "nginx",
								Listeners: []gwapiv1.Listener{{
									Name:     "api-example-com-http",
									Port:     80,
									Protocol: gwapiv1.HTTPProtocolType,
									Hostname: ptr.To(gwapiv1.Hostname("api.example.com")),
								}},
							},
						},
					},
				},
				HTTPRoutes: map[types.NamespacedName]providerir.HTTPRouteContext{
					{Namespace: "default", Name: "production-ingress-api-example-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "production-ingress-api-example-com", Namespace: "default"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "nginx",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"api.example.com"},
								Rules: []gwapiv1.HTTPRouteRule{
									{
										Matches: []gwapiv1.HTTPRouteMatch{{
											Path: &gwapiv1.HTTPPathMatch{
												Type:  &gPathPrefix,
												Value: ptr.To("/api"),
											},
										}},
										// Path "/api" has api-service-v1 from production and api-service-v2 from canary
										BackendRefs: []gwapiv1.HTTPBackendRef{
											{
												BackendRef: gwapiv1.BackendRef{
													BackendObjectReference: gwapiv1.BackendObjectReference{
														Name: "api-service-v1",
														Port: ptr.To(gwapiv1.PortNumber(80)),
													},
													Weight: ptr.To(int32(90)), // Production gets 90%
												},
											},
											{
												BackendRef: gwapiv1.BackendRef{
													BackendObjectReference: gwapiv1.BackendObjectReference{
														Name: "api-service-v2",
														Port: ptr.To(gwapiv1.PortNumber(80)),
													},
													Weight: ptr.To(int32(10)), // Canary gets 10%
												},
											},
										},
									},
									{
										Matches: []gwapiv1.HTTPRouteMatch{{
											Path: &gwapiv1.HTTPPathMatch{
												Type:  &gPathPrefix,
												Value: ptr.To("/admin"),
											},
										}},
										// Path "/admin" has admin-service from production and api-service-v1 from canary
										// Note: api-service-v1 appears in both rules but with different weights based on source!
										BackendRefs: []gwapiv1.HTTPBackendRef{
											{
												BackendRef: gwapiv1.BackendRef{
													BackendObjectReference: gwapiv1.BackendObjectReference{
														Name: "admin-service",
														Port: ptr.To(gwapiv1.PortNumber(80)),
													},
													Weight: ptr.To(int32(90)), // Production gets 90%
												},
											},
											{
												BackendRef: gwapiv1.BackendRef{
													BackendObjectReference: gwapiv1.BackendObjectReference{
														Name: "api-service-v1",
														Port: ptr.To(gwapiv1.PortNumber(80)),
													},
													Weight: ptr.To(int32(10)), // Canary gets 10%
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
			expectedErrors: field.ErrorList{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			provider := NewProvider(&i2gw.ProviderConf{})

			nginxProvider := provider.(*Provider)
			nginxProvider.storage.Ingresses = tc.ingresses

			ir, errs := provider.ToIR()

			if len(errs) != len(tc.expectedErrors) {
				t.Errorf("Expected %d errors, got %d: %+v", len(tc.expectedErrors), len(errs), errs)
			} else {
				for i, e := range errs {
					if errors.Is(e, tc.expectedErrors[i]) {
						t.Errorf("Unexpected error message at %d index. Got %s, want: %s", i, e, tc.expectedErrors[i])
					}
				}
			}

			if len(ir.HTTPRoutes) != len(tc.expectedIR.HTTPRoutes) {
				t.Errorf("Expected %d HTTPRoutes, got %d: %+v",
					len(tc.expectedIR.HTTPRoutes), len(ir.HTTPRoutes), ir.HTTPRoutes)
			} else {
				for i, gotHTTPRouteContext := range ir.HTTPRoutes {
					key := types.NamespacedName{Namespace: gotHTTPRouteContext.HTTPRoute.Namespace, Name: gotHTTPRouteContext.HTTPRoute.Name}
					wantHTTPRouteContext := tc.expectedIR.HTTPRoutes[key]
					wantHTTPRouteContext.HTTPRoute.SetGroupVersionKind(common.HTTPRouteGVK)
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
					wantGatewayContext.Gateway.SetGroupVersionKind(common.GatewayGVK)
					if !apiequality.Semantic.DeepEqual(gotGatewayContext.Gateway, wantGatewayContext.Gateway) {
						t.Errorf("Expected Gateway %s to be %+v\n Got: %+v\n Diff: %s", i, wantGatewayContext.Gateway, gotGatewayContext.Gateway, cmp.Diff(wantGatewayContext.Gateway, gotGatewayContext.Gateway))
					}
				}
			}
		})
	}
}
