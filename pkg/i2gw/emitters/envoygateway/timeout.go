package envoygateway_emitter

import (
	"fmt"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

func (e *Emitter) EmitTimeout(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		ctx.MergeExtensionFeature(emitterir.TimeoutFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.TimeoutFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			timeOutIR := ir.(*emitterir.TimeoutFeatureIR)

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.RouteRuleAllIndex && idx < len(ctx.Spec.Rules) {
				sectionName = ctx.Spec.Rules[idx].Name
			}
			backendTrafficPolicy := e.getOrBuildBackendTrafficPolicy(ctx, sectionName, idx)
			if backendTrafficPolicy.Spec.Timeout == nil {
				backendTrafficPolicy.Spec.Timeout = &egapiv1a1.Timeout{}
			}
			backendTrafficPolicy.Spec.Timeout.TCP = &egapiv1a1.TCPTimeout{
				ConnectTimeout: timeOutIR.Duration,
			}

			timeOutIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted Timeout annotations of ingress %s/%s",
				timeOutIR.GetSource().IngressNN.Namespace, timeOutIR.GetSource().IngressNN.Name),
				&ctx.HTTPRoute)
		}
	}
}
