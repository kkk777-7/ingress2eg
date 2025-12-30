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

func (e *Emitter) EmitRedirect(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		for idx, ir := range ctx.ExtensionFeatures[emitterir.RedirectFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			redirectIR := ir.(*emitterir.RedirectFeatureIR)

			nn := types.NamespacedName{
				Namespace: ctx.Namespace,
				Name:      ctx.Name,
			}
			route := gwResources.HTTPRoutes[nn]

			if len(route.Spec.ParentRefs) == 0 {
				notify(notifications.ErrorNotification, fmt.Sprintf("Cannot convert Redirect annotation for Ingress %s/%s: no parent refs found",
					redirectIR.GetSource().IngressNN.Namespace, redirectIR.GetSource().IngressNN.Name),
					&ctx.HTTPRoute)
				continue
			}

			gwNN := types.NamespacedName{
				Namespace: string(ptr.Deref(route.Spec.ParentRefs[0].Namespace, gwapiv1.Namespace(ctx.Namespace))),
				Name:      string(route.Spec.ParentRefs[0].Name),
			}
			gw := gwResources.Gateways[gwNN]

			e.emitSslRedirect(&route, &gw, redirectIR, idx)
			e.emitRedirect(&route, redirectIR, idx)

			gwResources.HTTPRoutes[nn] = route

			redirectIR.SetParsed()
			ruleInfo := ""
			if idx != emitterir.RouteRuleAllIndex && idx < len(ctx.Spec.Rules) && ctx.Spec.Rules[idx].Name != nil {
				ruleInfo = fmt.Sprintf(" for rule %s", *ctx.Spec.Rules[idx].Name)
			}
			notify(notifications.InfoNotification, fmt.Sprintf("converted Redirect annotations of ingress %s/%s%s",
				redirectIR.GetSource().IngressNN.Namespace, redirectIR.GetSource().IngressNN.Name, ruleInfo),
				&ctx.HTTPRoute)
		}
	}
}

func (e *Emitter) emitSslRedirect(route *gwapiv1.HTTPRoute, gw *gwapiv1.Gateway, redirectIR *emitterir.RedirectFeatureIR, ruleIdx int) {
	if redirectIR.Ssl == nil {
		return
	}

	if redirectIR.Ssl.Force {
		filter := gwapiv1.HTTPRequestRedirectFilter{
			Scheme:     ptr.To("https"),
			StatusCode: ptr.To(301),
		}
		route.Spec.Rules[ruleIdx].Filters = append(route.Spec.Rules[ruleIdx].Filters,
			gwapiv1.HTTPRouteFilter{
				Type:            gwapiv1.HTTPRouteFilterRequestRedirect,
				RequestRedirect: &filter,
			},
		)
	} else {
		var foundHTTPSListener bool
		for _, listener := range gw.Spec.Listeners {
			if listener.Protocol == gwapiv1.HTTPSProtocolType {
				foundHTTPSListener = true
				break
			}
		}

		if foundHTTPSListener {
			filter := gwapiv1.HTTPRequestRedirectFilter{
				Scheme:     ptr.To("https"),
				StatusCode: ptr.To(301),
			}
			route.Spec.Rules[ruleIdx].Filters = append(route.Spec.Rules[ruleIdx].Filters,
				gwapiv1.HTTPRouteFilter{
					Type:            gwapiv1.HTTPRouteFilterRequestRedirect,
					RequestRedirect: &filter,
				},
			)
		}
	}
}

func (e *Emitter) emitRedirect(route *gwapiv1.HTTPRoute, redirectIR *emitterir.RedirectFeatureIR, ruleIdx int) {
	if redirectIR.Url == "" {
		return
	}

	components, err := parseRedirectURL(redirectIR.Url)
	if err != nil {
		notify(notifications.ErrorNotification, fmt.Sprintf("Cannot convert Redirect annotation for Ingress %s/%s: %v",
			redirectIR.GetSource().IngressNN.Namespace, redirectIR.GetSource().IngressNN.Name, err),
			route)
		return
	}

	// Build the redirect filter
	filter := gwapiv1.HTTPRequestRedirectFilter{
		Scheme:     components.scheme,
		Hostname:   components.hostname,
		Port:       components.port,
		Path:       components.path,
		StatusCode: ptr.To(redirectIR.StatusCode),
	}

	route.Spec.Rules[ruleIdx].Filters = append(route.Spec.Rules[ruleIdx].Filters,
		gwapiv1.HTTPRouteFilter{
			Type:            gwapiv1.HTTPRouteFilterRequestRedirect,
			RequestRedirect: &filter,
		},
	)
}
