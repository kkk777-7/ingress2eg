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

func (e *Emitter) EmitBasicAuth(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		ctx.MergeExtensionFeature(emitterir.BasicAuthFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.BasicAuthFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			basicAuthIR := ir.(*emitterir.BasicAuthFeatureIR)

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.RouteRuleAllIndex && idx < len(ctx.Spec.Rules) {
				sectionName = ctx.Spec.Rules[idx].Name
			}

			securityPolicy := e.getOrBuildSecurityPolicy(ctx, sectionName, idx)
			securityPolicy.Spec.BasicAuth = &egapiv1a1.BasicAuth{
				Users: gwapiv1.SecretObjectReference{
					Group:     ptr.To(gwapiv1.Group(SecretGVK.Group)),
					Kind:      ptr.To(gwapiv1.Kind(SecretGVK.Kind)),
					Name:      gwapiv1.ObjectName(basicAuthIR.Name),
					Namespace: ptr.To(gwapiv1.Namespace(basicAuthIR.Namespace)),
				},
			}

			basicAuthIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted BasicAuth annotations of ingress %s/%s",
				basicAuthIR.GetSource().IngressNN.Namespace, basicAuthIR.GetSource().IngressNN.Name),
				&ctx.HTTPRoute)
		}
	}
}

func (e *Emitter) EmitMTLS(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.Gateways {
		for idx, ir := range ctx.ExtensionFeatures[emitterir.MTLSFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			mtlsIR := ir.(*emitterir.MTLSFeatureIR)

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.ListenerAllIndex && idx < len(ctx.Spec.Listeners) {
				sectionName = &ctx.Spec.Listeners[idx].Name
			}

			clientTrafficPolicy := e.getOrBuildClientTrafficPolicy(ctx, sectionName, idx)
			if clientTrafficPolicy.Spec.TLS == nil {
				clientTrafficPolicy.Spec.TLS = &egapiv1a1.ClientTLSSettings{}
			}
			if clientTrafficPolicy.Spec.TLS.ClientValidation == nil {
				clientTrafficPolicy.Spec.TLS.ClientValidation = &egapiv1a1.ClientValidationContext{}
			}
			clientTrafficPolicy.Spec.TLS.ClientValidation.CACertificateRefs = append(clientTrafficPolicy.Spec.TLS.ClientValidation.CACertificateRefs,
				gwapiv1.SecretObjectReference{
					Group:     ptr.To(gwapiv1.Group(SecretGVK.Group)),
					Kind:      ptr.To(gwapiv1.Kind(SecretGVK.Kind)),
					Name:      gwapiv1.ObjectName(mtlsIR.Name),
					Namespace: ptr.To(gwapiv1.Namespace(mtlsIR.Namespace)),
				},
			)

			mtlsIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted mTLS annotations of ingress %s/%s",
				mtlsIR.GetSource().IngressNN.Namespace, mtlsIR.GetSource().IngressNN.Name),
				&ctx.Gateway)
		}
	}
}
