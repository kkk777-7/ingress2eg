package envoygateway_emitter

import (
	"fmt"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

func (e *Emitter) EmitRateLimit(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		ctx.MergeExtensionFeature(emitterir.RateLimitFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.RateLimitFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			rateLimitIR := ir.(*emitterir.RateLimitFeatureIR)

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.RouteRuleAllIndex && idx < len(ctx.Spec.Rules) {
				sectionName = ctx.Spec.Rules[idx].Name
			}
			backendTrafficPolicy := e.getOrBuildBackendTrafficPolicy(ctx, sectionName, idx)

			if backendTrafficPolicy.Spec.RateLimit == nil {
				backendTrafficPolicy.Spec.RateLimit = &egapiv1a1.RateLimitSpec{}
			}
			if backendTrafficPolicy.Spec.RateLimit.Local == nil {
				backendTrafficPolicy.Spec.RateLimit.Local = &egapiv1a1.LocalRateLimit{}
			}

			var unit egapiv1a1.RateLimitUnit
			switch rateLimitIR.Unit {
			case emitterir.SecondTimeUnit:
				unit = egapiv1a1.RateLimitUnitSecond
			case emitterir.MinuteTimeUnit:
				unit = egapiv1a1.RateLimitUnitMinute
			default:
				// should not happen
			}

			backendTrafficPolicy.Spec.RateLimit.Local.Rules = append(backendTrafficPolicy.Spec.RateLimit.Local.Rules,
				egapiv1a1.RateLimitRule{
					Limit: egapiv1a1.RateLimitValue{
						Requests: rateLimitIR.LimitValue,
						Unit:     unit,
					},
					ClientSelectors: []egapiv1a1.RateLimitSelectCondition{
						{
							SourceCIDR: &egapiv1a1.SourceMatch{
								Type:  ptr.To(egapiv1a1.SourceMatchDistinct),
								Value: "0.0.0.0/0",
							},
						},
					},
				},
			)

			rateLimitIR.SetParsed()
			ruleInfo := ""
			if sectionName != nil {
				ruleInfo = fmt.Sprintf(" for rule %s", *sectionName)
			}
			notify(notifications.InfoNotification, fmt.Sprintf("converted RateLimit annotations of ingress %s/%s%s",
				rateLimitIR.GetSource().IngressNN.Namespace, rateLimitIR.GetSource().IngressNN.Name, ruleInfo),
				&ctx.HTTPRoute)
		}
	}
}
