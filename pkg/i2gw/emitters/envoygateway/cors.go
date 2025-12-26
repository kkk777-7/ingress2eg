package envoygateway_emitter

import (
	"fmt"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"github.com/samber/lo"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

func (e *Emitter) EmitCors(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		ctx.MergeExtensionFeature(emitterir.CORSFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.CORSFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			corsIR := ir.(*emitterir.CORSFeatureIR)

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.RouteRuleAllIndex && idx < len(ctx.Spec.Rules) {
				sectionName = ctx.Spec.Rules[idx].Name
			}
			securityPolicy := e.getOrBuildSecurityPolicy(ctx, sectionName, idx)
			if securityPolicy.Spec.CORS == nil {
				securityPolicy.Spec.CORS = &egapiv1a1.CORS{}
			}
			if len(corsIR.AllowOrigins) > 0 {
				securityPolicy.Spec.CORS.AllowOrigins = lo.Map(corsIR.AllowOrigins, func(a string, _ int) egapiv1a1.Origin {
					return egapiv1a1.Origin(a)
				})
			}
			if len(corsIR.AllowMethods) > 0 {
				securityPolicy.Spec.CORS.AllowMethods = corsIR.AllowMethods
			}
			if len(corsIR.AllowHeaders) > 0 {
				securityPolicy.Spec.CORS.AllowHeaders = corsIR.AllowHeaders
			}
			if len(corsIR.ExposeHeaders) > 0 {
				securityPolicy.Spec.CORS.ExposeHeaders = corsIR.ExposeHeaders
			}
			if corsIR.MaxAge != nil {
				securityPolicy.Spec.CORS.MaxAge = corsIR.MaxAge
			}
			if corsIR.AllowCredentials != nil {
				securityPolicy.Spec.CORS.AllowCredentials = corsIR.AllowCredentials
			}

			corsIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted CORS annotations of ingress %s/%s",
				corsIR.GetSource().IngressNN.Namespace, corsIR.GetSource().IngressNN.Name),
				&ctx.HTTPRoute)
		}
	}
}
