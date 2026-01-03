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
	for _, ctx := range ir.HTTPRoutes {
		ctx.MergeExtensionFeature(emitterir.BufferFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.BufferFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			bufferIR := ir.(*emitterir.BufferFeatureIR)

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.RouteRuleAllIndex && idx < len(ctx.Spec.Rules) {
				sectionName = ctx.Spec.Rules[idx].Name
			}
			backendTrafficPolicy := e.getOrBuildBackendTrafficPolicy(ctx, sectionName, idx)
			quantity := resource.MustParse(bufferIR.LimitValue)
			backendTrafficPolicy.Spec.RequestBuffer = &egapiv1a1.RequestBuffer{
				Limit: quantity,
			}

			bufferIR.SetParsed()
			ruleInfo := ""
			if sectionName != nil {
				ruleInfo = fmt.Sprintf(" for rule %s", *sectionName)
			}
			notify(notifications.InfoNotification, fmt.Sprintf("converted Buffer annotations of ingress %s/%s%s",
				bufferIR.GetSource().IngressNN.Namespace, bufferIR.GetSource().IngressNN.Name, ruleInfo),
				&ctx.HTTPRoute)
		}
	}
}
