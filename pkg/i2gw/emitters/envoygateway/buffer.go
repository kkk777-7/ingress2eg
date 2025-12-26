package envoygateway_emitter

import (
	"fmt"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

func (e *Emitter) EmitBuffer(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.Gateways {
		ctx.MergeExtensionFeature(emitterir.BufferFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.BufferFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			bufferIR := ir.(*emitterir.BufferFeatureIR)

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.ListenerAllIndex && idx < len(ctx.Spec.Listeners) {
				sectionName = &ctx.Spec.Listeners[idx].Name
			}

			clientTrafficPolicy := e.getOrBuildClientTrafficPolicy(ctx, sectionName, idx)
			if clientTrafficPolicy.Spec.Connection == nil {
				clientTrafficPolicy.Spec.Connection = &egapiv1a1.ClientConnection{}
			}
			quantity := resource.MustParse(bufferIR.LimitValue)
			clientTrafficPolicy.Spec.Connection.BufferLimit = &quantity

			bufferIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted Buffer annotations of ingress %s/%s",
				bufferIR.GetSource().IngressNN.Namespace, bufferIR.GetSource().IngressNN.Name),
				&ctx.Gateway)
		}
	}
}
