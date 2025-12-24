package envoygateway_emitter

import (
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

func (e *Emitter) EmitRegex(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		for idx, regexIR := range ctx.ExtensionFeatures[emitterir.RegexFeatureKey] {
			if regexIR.IsParsed() {
				continue
			}
			nn := types.NamespacedName{
				Namespace: ctx.Namespace,
				Name:      ctx.Name,
			}

			route := gwResources.HTTPRoutes[nn]
			for i := range route.Spec.Rules[idx].Matches {
				if route.Spec.Rules[idx].Matches[i].Path != nil {
					route.Spec.Rules[idx].Matches[i].Path.Type = ptr.To(gwapiv1.PathMatchRegularExpression)
				}
			}
			gwResources.HTTPRoutes[nn] = route

			regexIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted Regex annotations of ingress %s/%s",
				regexIR.GetSource().IngressNN.Namespace, regexIR.GetSource().IngressNN.Name),
				&ctx.HTTPRoute)
		}
	}
}
