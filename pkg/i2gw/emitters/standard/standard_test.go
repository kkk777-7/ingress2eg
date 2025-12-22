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

package standard_emitter

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var (
	GatewayGVK = schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "Gateway",
	}

	HTTPRouteGVK = schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	}
)

func Test_ToGatewayResources(t *testing.T) {
	gPathPrefix := gwapiv1.PathMatchPathPrefix

	testCases := []struct {
		desc                     string
		ir                       emitterir.EmitterIR
		expectedGatewayResources i2gw.GatewayResources
		expectedErrors           field.ErrorList
	}{
		{
			desc:                     "empty",
			ir:                       emitterir.EmitterIR{},
			expectedGatewayResources: i2gw.GatewayResources{},
			expectedErrors:           field.ErrorList{},
		},
		{
			desc: "no additional extensions",
			ir: emitterir.EmitterIR{
				Gateways: map[types.NamespacedName]emitterir.GatewayContext{
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
				HTTPRoutes: map[types.NamespacedName]emitterir.HTTPRouteContext{
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
			expectedGatewayResources: i2gw.GatewayResources{
				Gateways: map[types.NamespacedName]gwapiv1.Gateway{
					{Namespace: "test", Name: "simple"}: {
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
				HTTPRoutes: map[types.NamespacedName]gwapiv1.HTTPRoute{
					{Namespace: "test", Name: "simple-example-com"}: {
						ObjectMeta: metav1.ObjectMeta{Name: "simple-example-com", Namespace: "test"},
						Spec: gwapiv1.HTTPRouteSpec{
							CommonRouteSpec: gwapiv1.CommonRouteSpec{
								ParentRefs: []gwapiv1.ParentReference{{
									Name: "simple",
								}},
							},
							Hostnames: []gwapiv1.Hostname{"example.com"},
							Rules: []gwapiv1.HTTPRouteRule{{
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
		{
			desc: "duplicated backends",
			ir: emitterir.EmitterIR{
				Gateways: map[types.NamespacedName]emitterir.GatewayContext{
					{Namespace: "test", Name: "example-proxy"}: {
						Gateway: gwapiv1.Gateway{
							ObjectMeta: metav1.ObjectMeta{Name: "example-proxy", Namespace: "test"},
							Spec: gwapiv1.GatewaySpec{
								GatewayClassName: "example-proxy",
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
				HTTPRoutes: map[types.NamespacedName]emitterir.HTTPRouteContext{
					{Namespace: "test", Name: "duplicate-example-com"}: {
						HTTPRoute: gwapiv1.HTTPRoute{
							ObjectMeta: metav1.ObjectMeta{Name: "duplicate-example-com", Namespace: "test"},
							Spec: gwapiv1.HTTPRouteSpec{
								CommonRouteSpec: gwapiv1.CommonRouteSpec{
									ParentRefs: []gwapiv1.ParentReference{{
										Name: "example-proxy",
									}},
								},
								Hostnames: []gwapiv1.Hostname{"example.com"},
								Rules: []gwapiv1.HTTPRouteRule{{
									Matches: []gwapiv1.HTTPRouteMatch{{
										Path: &gwapiv1.HTTPPathMatch{
											Type:  &gPathPrefix,
											Value: ptr.To("/foo"),
										},
									}},
									BackendRefs: []gwapiv1.HTTPBackendRef{
										{
											BackendRef: gwapiv1.BackendRef{
												BackendObjectReference: gwapiv1.BackendObjectReference{
													Name: "example",
													Port: ptr.To(gwapiv1.PortNumber(3000)),
												},
											},
										},
										{
											BackendRef: gwapiv1.BackendRef{
												BackendObjectReference: gwapiv1.BackendObjectReference{
													Name: "example",
													Port: ptr.To(gwapiv1.PortNumber(3000)),
												},
											},
										},
									},
								}},
							},
						},
					},
				},
			},
			expectedGatewayResources: i2gw.GatewayResources{
				Gateways: map[types.NamespacedName]gwapiv1.Gateway{
					{Namespace: "test", Name: "example-proxy"}: {
						TypeMeta: metav1.TypeMeta{
							Kind:       "Gateway",
							APIVersion: "gateway.networking.k8s.io/v1",
						},
						ObjectMeta: metav1.ObjectMeta{Name: "example-proxy", Namespace: "test"},
						Spec: gwapiv1.GatewaySpec{
							GatewayClassName: "example-proxy",
							Listeners: []gwapiv1.Listener{{
								Name:     "example-com-http",
								Port:     80,
								Protocol: gwapiv1.HTTPProtocolType,
								Hostname: ptr.To(gwapiv1.Hostname("example.com")),
							}},
						},
					},
				},
				HTTPRoutes: map[types.NamespacedName]gwapiv1.HTTPRoute{
					{Namespace: "test", Name: "duplicate-example-com"}: {
						TypeMeta: metav1.TypeMeta{
							Kind:       "HTTPRoute",
							APIVersion: "gateway.networking.k8s.io/v1",
						},
						ObjectMeta: metav1.ObjectMeta{Name: "duplicate-example-com", Namespace: "test"},
						Spec: gwapiv1.HTTPRouteSpec{
							CommonRouteSpec: gwapiv1.CommonRouteSpec{
								ParentRefs: []gwapiv1.ParentReference{{
									Name: "example-proxy",
								}},
							},
							Hostnames: []gwapiv1.Hostname{"example.com"},
							Rules: []gwapiv1.HTTPRouteRule{{
								Matches: []gwapiv1.HTTPRouteMatch{{
									Path: &gwapiv1.HTTPPathMatch{
										Type:  &gPathPrefix,
										Value: ptr.To("/foo"),
									},
								}},
								BackendRefs: []gwapiv1.HTTPBackendRef{
									{
										BackendRef: gwapiv1.BackendRef{
											BackendObjectReference: gwapiv1.BackendObjectReference{
												Name: "example",
												Port: ptr.To(gwapiv1.PortNumber(3000)),
											},
										},
									},
								},
							}},
						},
						Status: gwapiv1.HTTPRouteStatus{
							RouteStatus: gwapiv1.RouteStatus{
								Parents: []gwapiv1.RouteParentStatus{},
							},
						},
					},
				},
			},
			expectedErrors: field.ErrorList{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			emitter := Emitter{}
			gatewayResouces, errs := emitter.Emit(tc.ir)

			if len(errs) != len(tc.expectedErrors) {
				t.Errorf("Expected %d errors, got %d: %+v", len(tc.expectedErrors), len(errs), errs)
			} else {
				for i, e := range errs {
					if errors.Is(e, tc.expectedErrors[i]) {
						t.Errorf("Unexpected error message at %d index. Got %s, want: %s", i, e, tc.expectedErrors[i])
					}
				}
			}

			if len(gatewayResouces.HTTPRoutes) != len(tc.expectedGatewayResources.HTTPRoutes) {
				t.Errorf("Expected %d HTTPRoutes, got %d: %+v",
					len(tc.expectedGatewayResources.HTTPRoutes), len(gatewayResouces.HTTPRoutes), gatewayResouces.HTTPRoutes)
			} else {
				for i, got := range gatewayResouces.HTTPRoutes {
					got.SetGroupVersionKind(HTTPRouteGVK)
					key := types.NamespacedName{Namespace: got.Namespace, Name: got.Name}
					want := tc.expectedGatewayResources.HTTPRoutes[key]
					want.SetGroupVersionKind(HTTPRouteGVK)
					if !apiequality.Semantic.DeepEqual(got, want) {
						t.Errorf("Expected HTTPRoute %s to be %+v\n Got: %+v\n Diff: %s", i, want, got, cmp.Diff(want, got))
					}
				}
			}

			if len(gatewayResouces.Gateways) != len(tc.expectedGatewayResources.Gateways) {
				t.Errorf("Expected %d Gateways, got %d: %+v",
					len(tc.expectedGatewayResources.Gateways), len(gatewayResouces.Gateways), gatewayResouces.Gateways)
			} else {
				for i, got := range gatewayResouces.Gateways {
					got.SetGroupVersionKind(GatewayGVK)
					key := types.NamespacedName{Namespace: got.Namespace, Name: got.Name}
					want := tc.expectedGatewayResources.Gateways[key]
					want.SetGroupVersionKind(GatewayGVK)
					if !apiequality.Semantic.DeepEqual(got, want) {
						t.Errorf("Expected Gateway %s to be %+v\n Got: %+v\n Diff: %s", i, want, got, cmp.Diff(want, got))
					}
				}
			}
		})
	}
}
