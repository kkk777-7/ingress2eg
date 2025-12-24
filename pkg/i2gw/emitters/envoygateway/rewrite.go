package envoygateway_emitter

import (
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

// convertNginxSubstitutionToEnvoy converts nginx rewrite-target format ($1, $2) to Envoy Gateway format (\1, \2)
func convertNginxSubstitutionToEnvoy(nginxSubstitution string) string {
	// Replace $1, $2, etc. with \1, \2, etc.
	re := regexp.MustCompile(`\$(\d+)`)
	return re.ReplaceAllString(nginxSubstitution, `\$1`)
}

func (c *Emitter) EmitRewrite(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		for idx, ir := range ctx.ExtensionFeatures[emitterir.RewriteFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			rewriteIR := ir.(*emitterir.RewriteFeatureIR)

			nn := types.NamespacedName{
				Namespace: ctx.Namespace,
				Name:      ctx.Name,
			}
			route := gwResources.HTTPRoutes[nn]

			regexIR := ctx.ExtensionFeatures[emitterir.RegexFeatureKey][idx]
			if regexIR == nil {
				filter := gwapiv1.HTTPURLRewriteFilter{
					Path: ptr.To(gwapiv1.HTTPPathModifier{
						Type:            gwapiv1.FullPathHTTPPathModifier,
						ReplaceFullPath: ptr.To(rewriteIR.Target),
					}),
				}
				route.Spec.Rules[idx].Filters = append(route.Spec.Rules[idx].Filters,
					gwapiv1.HTTPRouteFilter{
						Type:       gwapiv1.HTTPRouteFilterURLRewrite,
						URLRewrite: &filter,
					},
				)
			} else {
				regexIR := regexIR.(*emitterir.RegexFeatureIR)
				if regexIR.PathPattern == "" {
					notify(notifications.ErrorNotification, fmt.Sprintf("Cannot convert Rewrite annotation for Ingress %s/%s: regex path match is required",
						rewriteIR.GetSource().IngressNN.Namespace, rewriteIR.GetSource().IngressNN.Name),
						&ctx.HTTPRoute)
					continue
				}

				filter := builderRewriteHTTPRouteFilter(ctx, idx, regexIR, rewriteIR)

				obj, err := i2gw.CastToUnstructured(filter)
				if err != nil {
					notify(notifications.ErrorNotification, "Failed to cast HTTPRouteFilter to unstructured", filter)
					continue
				}
				gwResources.GatewayExtensions = append(gwResources.GatewayExtensions, *obj)

				// Set HTTPRoute path to "/" since HTTPRouteFilter will handle the regex matching
				route.Spec.Rules[idx].Matches[0].Path = &gwapiv1.HTTPPathMatch{
					Type:  ptr.To(gwapiv1.PathMatchPathPrefix),
					Value: ptr.To("/"),
				}
				route.Spec.Rules[idx].Filters = append(route.Spec.Rules[idx].Filters,
					gwapiv1.HTTPRouteFilter{
						Type: gwapiv1.HTTPRouteFilterExtensionRef,
						ExtensionRef: &gwapiv1.LocalObjectReference{
							Group: gwapiv1.Group(HTTPRouteFilterGVK.Group),
							Kind:  gwapiv1.Kind(HTTPRouteFilterGVK.Kind),
							Name:  gwapiv1.ObjectName(filter.Name),
						},
					},
				)
			}

			gwResources.HTTPRoutes[nn] = route

			rewriteIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted Rewrite annotations of ingress %s/%s",
				rewriteIR.GetSource().IngressNN.Namespace, rewriteIR.GetSource().IngressNN.Name),
				&ctx.HTTPRoute)
		}
	}
}
