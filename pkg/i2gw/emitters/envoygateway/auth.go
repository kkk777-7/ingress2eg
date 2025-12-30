package envoygateway_emitter

import (
	"fmt"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"

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
			ruleInfo := ""
			if sectionName != nil {
				ruleInfo = fmt.Sprintf(" for rule %s", *sectionName)
			}
			notify(notifications.InfoNotification, fmt.Sprintf("converted BasicAuth annotations of ingress %s/%s%s",
				basicAuthIR.GetSource().IngressNN.Namespace, basicAuthIR.GetSource().IngressNN.Name, ruleInfo),
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
			if clientTrafficPolicy.Namespace != mtlsIR.Namespace {
				refgrant := &gwapiv1b1.ReferenceGrant{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-mtls", clientTrafficPolicy.Name),
						Namespace: mtlsIR.Namespace,
					},
					Spec: gwapiv1b1.ReferenceGrantSpec{
						From: []gwapiv1b1.ReferenceGrantFrom{
							{
								Group:     gwapiv1b1.Group(SecurityPolicyGVK.Group),
								Kind:      gwapiv1.Kind(SecurityPolicyGVK.Kind),
								Namespace: gwapiv1b1.Namespace(clientTrafficPolicy.Namespace),
							},
						},
						To: []gwapiv1b1.ReferenceGrantTo{
							{
								Group: gwapiv1b1.Group(SecretGVK.Group),
								Kind:  gwapiv1b1.Kind(SecretGVK.Kind),
								Name:  ptr.To(gwapiv1b1.ObjectName(mtlsIR.Name)),
							},
						},
					},
				}
				refgrant.SetGroupVersionKind(ReferenceGrantGVK)
				gwResources.ReferenceGrants[types.NamespacedName{Namespace: refgrant.Namespace, Name: refgrant.Name}] = *refgrant
			}

			mtlsIR.SetParsed()
			listenerInfo := ""
			if sectionName != nil {
				listenerInfo = fmt.Sprintf(" for listener %s", *sectionName)
			}
			notify(notifications.InfoNotification, fmt.Sprintf("converted mTLS annotations of ingress %s/%s%s",
				mtlsIR.GetSource().IngressNN.Namespace, mtlsIR.GetSource().IngressNN.Name, listenerInfo),
				&ctx.Gateway)
		}
	}
}

func (e *Emitter) EmitExternalAuth(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		ctx.MergeExtensionFeature(emitterir.ExternalAuthFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.ExternalAuthFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			externalAuthIR := ir.(*emitterir.ExternalAuthFeatureIR)

			backendRef, path, err := parseK8sServiceURL(externalAuthIR.Url, ctx.Namespace)
			if err != nil {
				notify(notifications.ErrorNotification, fmt.Sprintf("Failed to parse ExternalAuth URL for Ingress %s/%s, Only Kubernetes service domains are supported: %v",
					externalAuthIR.GetSource().IngressNN.Namespace, externalAuthIR.GetSource().IngressNN.Name, err),
					&ctx.HTTPRoute)
				continue
			}

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.RouteRuleAllIndex && idx < len(ctx.Spec.Rules) {
				sectionName = ctx.Spec.Rules[idx].Name
			}
			securityPolicy := e.getOrBuildSecurityPolicy(ctx, sectionName, idx)

			securityPolicy.Spec.ExtAuth = &egapiv1a1.ExtAuth{
				HTTP: &egapiv1a1.HTTPExtAuthService{
					BackendCluster: egapiv1a1.BackendCluster{
						BackendRefs: []egapiv1a1.BackendRef{
							{
								BackendObjectReference: *backendRef,
							},
						},
					},
					Path:             path,
					HeadersToBackend: externalAuthIR.AllowedResponseHeaders,
				},
			}

			externalAuthIR.SetParsed()
			ruleInfo := ""
			if sectionName != nil {
				ruleInfo = fmt.Sprintf(" for rule %s", *sectionName)
			}
			notify(notifications.InfoNotification, fmt.Sprintf("converted ExternalAuth annotations of ingress %s/%s%s",
				externalAuthIR.GetSource().IngressNN.Namespace, externalAuthIR.GetSource().IngressNN.Name, ruleInfo),
				&ctx.HTTPRoute)
		}
	}
}
