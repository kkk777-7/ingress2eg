package envoygateway_emitter

import "k8s.io/apimachinery/pkg/runtime/schema"

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

	ReferenceGrantGVK = schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1beta1",
		Kind:    "ReferenceGrant",
	}

	SecretGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}

	HTTPRouteFilterGVK = schema.GroupVersionKind{
		Group:   "gateway.envoyproxy.io",
		Version: "v1alpha1",
		Kind:    "HTTPRouteFilter",
	}

	BackendGVK = schema.GroupVersionKind{
		Group:   "gateway.envoyproxy.io",
		Version: "v1alpha1",
		Kind:    "Backend",
	}

	SecurityPolicyGVK = schema.GroupVersionKind{
		Group:   "gateway.envoyproxy.io",
		Version: "v1alpha1",
		Kind:    "SecurityPolicy",
	}

	ClientTrafficPolicyGVK = schema.GroupVersionKind{
		Group:   "gateway.envoyproxy.io",
		Version: "v1alpha1",
		Kind:    "ClientTrafficPolicy",
	}

	BackendTrafficPolicyGVK = schema.GroupVersionKind{
		Group:   "gateway.envoyproxy.io",
		Version: "v1alpha1",
		Kind:    "BackendTrafficPolicy",
	}
)
