package envoygateway_emitter

import (
	"fmt"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

func (e *Emitter) EmitAffinity(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		ctx.MergeExtensionFeature(emitterir.AffinityFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.AffinityFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			affinityIR := ir.(*emitterir.AffinityFeatureIR)

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.RouteRuleAllIndex && idx < len(ctx.Spec.Rules) {
				sectionName = ctx.Spec.Rules[idx].Name
			}
			backendTrafficPolicy := e.getOrBuildBackendTrafficPolicy(ctx, sectionName, idx)
			if backendTrafficPolicy.Spec.LoadBalancer == nil {
				backendTrafficPolicy.Spec.LoadBalancer = &egapiv1a1.LoadBalancer{}
			}
			backendTrafficPolicy.Spec.LoadBalancer.Type = egapiv1a1.ConsistentHashLoadBalancerType
			backendTrafficPolicy.Spec.LoadBalancer.ConsistentHash = &egapiv1a1.ConsistentHash{
				Type: egapiv1a1.CookieConsistentHashType,
				Cookie: &egapiv1a1.Cookie{
					Name: affinityIR.CookieName,
					TTL:  affinityIR.CookieTTLSeconds,
				},
			}
			if affinityIR.CookieSameSite != "" {
				if backendTrafficPolicy.Spec.LoadBalancer.ConsistentHash.Cookie.Attributes == nil {
					backendTrafficPolicy.Spec.LoadBalancer.ConsistentHash.Cookie.Attributes = make(map[string]string)
				}
				backendTrafficPolicy.Spec.LoadBalancer.ConsistentHash.Cookie.Attributes["SameSite"] = affinityIR.CookieSameSite
			}

			affinityIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted Affinity annotations of ingress %s/%s",
				affinityIR.GetSource().IngressNN.Namespace, affinityIR.GetSource().IngressNN.Name),
				&ctx.HTTPRoute)
		}
	}
}
