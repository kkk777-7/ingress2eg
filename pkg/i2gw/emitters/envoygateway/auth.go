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
			if idx != -1 {
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
