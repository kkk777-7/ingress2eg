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
	SecurityPolicies      map[types.NamespacedName]*egapiv1a1.SecurityPolicy
	ClientTrafficPolicies map[types.NamespacedName]*egapiv1a1.ClientTrafficPolicy
}

func NewBuilderMap() *BuilderMap {
	return &BuilderMap{
		SecurityPolicies:      make(map[types.NamespacedName]*egapiv1a1.SecurityPolicy),
		ClientTrafficPolicies: make(map[types.NamespacedName]*egapiv1a1.ClientTrafficPolicy),
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
