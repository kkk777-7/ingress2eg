package envoygateway_emitter

import "k8s.io/apimachinery/pkg/runtime/schema"

var (
	HTTPRouteFilterGVK = schema.GroupVersionKind{
		Group:   "gateway.envoyproxy.io",
		Version: "v1alpha1",
		Kind:    "HTTPRouteFilter",
	}
)
