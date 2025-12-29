package envoygateway_emitter

import (
	"fmt"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
)

type BuilderMap struct {
	Backends               map[types.NamespacedName]*egapiv1a1.Backend
	SecurityPolicies       map[types.NamespacedName]*egapiv1a1.SecurityPolicy
	ClientTrafficPolicies  map[types.NamespacedName]*egapiv1a1.ClientTrafficPolicy
	BackendTrafficPolicies map[types.NamespacedName]*egapiv1a1.BackendTrafficPolicy
}

func NewBuilderMap() *BuilderMap {
	return &BuilderMap{
		Backends:               make(map[types.NamespacedName]*egapiv1a1.Backend),
		SecurityPolicies:       make(map[types.NamespacedName]*egapiv1a1.SecurityPolicy),
		ClientTrafficPolicies:  make(map[types.NamespacedName]*egapiv1a1.ClientTrafficPolicy),
		BackendTrafficPolicies: make(map[types.NamespacedName]*egapiv1a1.BackendTrafficPolicy),
	}
}

func buildRewriteHTTPRouteFilter(
	ctx emitterir.HTTPRouteContext,
	ruleIdx int,
	regexIR *emitterir.RegexFeatureIR,
	rewriteIR *emitterir.RewriteFeatureIR,
) *egapiv1a1.HTTPRouteFilter {
	filter := &egapiv1a1.HTTPRouteFilter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", ctx.Name, ruleIdx),
			Namespace: ctx.Namespace,
		},
		Spec: egapiv1a1.HTTPRouteFilterSpec{
			URLRewrite: &egapiv1a1.HTTPURLRewriteFilter{
				Path: &egapiv1a1.HTTPPathModifier{
					Type: egapiv1a1.RegexHTTPPathModifier,
					ReplaceRegexMatch: &egapiv1a1.ReplaceRegexMatch{
						Pattern:      fmt.Sprintf("^%s$", regexIR.PathPattern),
						Substitution: convertNginxSubstitutionToEnvoy(rewriteIR.Target),
					},
				},
			},
		},
	}
	filter.SetGroupVersionKind(HTTPRouteFilterGVK)
	return filter
}

func (e *Emitter) getOrBuildBackend(routeNN types.NamespacedName, ruleIdx int, backendRef gwapiv1.BackendObjectReference) *egapiv1a1.Backend {
	name := fmt.Sprintf("%s-%d-%s", routeNN.Name, ruleIdx, backendRef.Name)
	if ruleIdx == emitterir.RouteRuleAllIndex {
		name = fmt.Sprintf("%s-%s", routeNN.Name, backendRef.Name)
	}
	key := types.NamespacedName{
		Name:      name,
		Namespace: routeNN.Namespace,
	}
	backend, exist := e.builderMap.Backends[key]
	if exist {
		return backend
	}

	backend = &egapiv1a1.Backend{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: routeNN.Namespace,
		},
		Spec: egapiv1a1.BackendSpec{
			Endpoints: []egapiv1a1.BackendEndpoint{
				{
					FQDN: &egapiv1a1.FQDNEndpoint{
						Hostname: fmt.Sprintf("%s.%s.svc.cluster.local", backendRef.Name, routeNN.Namespace),
						Port:     *backendRef.Port,
					},
				},
			},
		},
	}
	backend.SetGroupVersionKind(BackendGVK)

	e.builderMap.Backends[key] = backend
	return backend
}

func (e *Emitter) getOrBuildSecurityPolicy(ctx emitterir.HTTPRouteContext, sectionName *gwapiv1.SectionName, ruleIdx int) *egapiv1a1.SecurityPolicy {
	name := fmt.Sprintf("%s-%d", ctx.Name, ruleIdx)
	if ruleIdx == emitterir.RouteRuleAllIndex {
		name = ctx.Name
	}
	key := types.NamespacedName{
		Name:      name,
		Namespace: ctx.Namespace,
	}
	policy, exist := e.builderMap.SecurityPolicies[key]
	if exist {
		return policy
	}

	securityPolicy := &egapiv1a1.SecurityPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ctx.Namespace,
		},
		Spec: egapiv1a1.SecurityPolicySpec{
			PolicyTargetReferences: egapiv1a1.PolicyTargetReferences{
				TargetRefs: []gwapiv1.LocalPolicyTargetReferenceWithSectionName{
					{
						LocalPolicyTargetReference: gwapiv1.LocalPolicyTargetReference{
							Group: gwapiv1.Group(HTTPRouteGVK.Group),
							Kind:  gwapiv1.Kind(HTTPRouteGVK.Kind),
							Name:  gwapiv1.ObjectName(ctx.Name),
						},
						SectionName: sectionName,
					},
				},
			},
		},
	}
	securityPolicy.SetGroupVersionKind(SecurityPolicyGVK)

	e.builderMap.SecurityPolicies[key] = securityPolicy
	return securityPolicy
}

func (e *Emitter) getOrBuildClientTrafficPolicy(ctx emitterir.GatewayContext, sectionName *gwapiv1.SectionName, listenerIdx int) *egapiv1a1.ClientTrafficPolicy {
	name := fmt.Sprintf("%s-%d", ctx.Name, listenerIdx)
	if listenerIdx == emitterir.ListenerAllIndex {
		name = ctx.Name
	}
	key := types.NamespacedName{
		Name:      name,
		Namespace: ctx.Namespace,
	}
	policy, exist := e.builderMap.ClientTrafficPolicies[key]
	if exist {
		return policy
	}

	clientTrafficPolicy := &egapiv1a1.ClientTrafficPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ctx.Namespace,
		},
		Spec: egapiv1a1.ClientTrafficPolicySpec{
			PolicyTargetReferences: egapiv1a1.PolicyTargetReferences{
				TargetRefs: []gwapiv1.LocalPolicyTargetReferenceWithSectionName{
					{
						LocalPolicyTargetReference: gwapiv1.LocalPolicyTargetReference{
							Group: gwapiv1.Group(GatewayGVK.Group),
							Kind:  gwapiv1.Kind(GatewayGVK.Kind),
							Name:  gwapiv1.ObjectName(ctx.Name),
						},
						SectionName: sectionName,
					},
				},
			},
		},
	}
	clientTrafficPolicy.SetGroupVersionKind(ClientTrafficPolicyGVK)

	e.builderMap.ClientTrafficPolicies[key] = clientTrafficPolicy
	return clientTrafficPolicy
}

func (e *Emitter) getOrBuildBackendTrafficPolicy(ctx emitterir.HTTPRouteContext, sectionName *gwapiv1.SectionName, ruleIdx int) *egapiv1a1.BackendTrafficPolicy {
	name := fmt.Sprintf("%s-%d", ctx.Name, ruleIdx)
	if ruleIdx == emitterir.RouteRuleAllIndex {
		name = ctx.Name
	}
	key := types.NamespacedName{
		Name:      name,
		Namespace: ctx.Namespace,
	}
	policy, exist := e.builderMap.BackendTrafficPolicies[key]
	if exist {
		return policy
	}

	backendTrafficPolicy := &egapiv1a1.BackendTrafficPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ctx.Namespace,
		},
		Spec: egapiv1a1.BackendTrafficPolicySpec{
			PolicyTargetReferences: egapiv1a1.PolicyTargetReferences{
				TargetRefs: []gwapiv1.LocalPolicyTargetReferenceWithSectionName{
					{
						LocalPolicyTargetReference: gwapiv1.LocalPolicyTargetReference{
							Group: gwapiv1.Group(HTTPRouteGVK.Group),
							Kind:  gwapiv1.Kind(HTTPRouteGVK.Kind),
							Name:  gwapiv1.ObjectName(ctx.Name),
						},
						SectionName: sectionName,
					},
				},
			},
		},
	}
	backendTrafficPolicy.SetGroupVersionKind(BackendTrafficPolicyGVK)

	e.builderMap.BackendTrafficPolicies[key] = backendTrafficPolicy
	return backendTrafficPolicy
}
