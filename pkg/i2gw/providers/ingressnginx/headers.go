package ingressnginx

import (
	"fmt"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
	providerir "github.com/kkk777-7/ingress2eg/pkg/i2gw/provider_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/providers/common"
)

const (
	forwardedPrefixAnnotation = "nginx.ingress.kubernetes.io/x-forwarded-prefix"
	upstreamvHostAnnotation   = "nginx.ingress.kubernetes.io/upstream-vhost"
)

func headerFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
	ruleGroups := common.GetRuleGroups(ingresses)
	var errList field.ErrorList

	for _, rg := range ruleGroups {
		key := types.NamespacedName{Namespace: rg.Namespace, Name: common.RouteName(rg.Name, rg.Host)}

		// Get RuleBackendSources from Provider IR
		providerHTTPRouteContext, ok := pir.HTTPRoutes[key]
		if !ok {
			continue
		}

		emitterHTTPRouteContext, ok := eir.HTTPRoutes[key]
		if !ok {
			continue
		}

		// Track which Ingresses have header modification and their annotation keys
		ingressesWithHeaderMod := make(map[types.NamespacedName]sets.Set[string])

		for ruleIdx, backendSources := range providerHTTPRouteContext.RuleBackendSources {
			if ruleIdx >= len(emitterHTTPRouteContext.Spec.Rules) {
				errList = append(errList, field.InternalError(
					field.NewPath("httproute", emitterHTTPRouteContext.HTTPRoute.Name, "spec", "rules").Index(ruleIdx),
					fmt.Errorf("rule index %d exceeds available rules", ruleIdx),
				))
				continue
			}

			for _, source := range backendSources {
				if source.Ingress == nil {
					continue
				}
				ingress := *source.Ingress

				if val := ingress.Annotations[forwardedPrefixAnnotation]; val != "" {
					emitterHTTPRouteContext.Spec.Rules[ruleIdx].Filters = append(emitterHTTPRouteContext.Spec.Rules[ruleIdx].Filters,
						gwapiv1.HTTPRouteFilter{
							Type: gwapiv1.HTTPRouteFilterRequestHeaderModifier,
							RequestHeaderModifier: &gwapiv1.HTTPHeaderFilter{
								Add: []gwapiv1.HTTPHeader{
									{
										Name:  "X-Forwarded-Prefix",
										Value: val,
									},
								},
							},
						},
					)

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressesWithHeaderMod[ingressNN] == nil {
						ingressesWithHeaderMod[ingressNN] = sets.New[string]()
					}
					ingressesWithHeaderMod[ingressNN].Insert("x-forwarded-prefix")
				}

				if val := ingress.Annotations[upstreamvHostAnnotation]; val != "" {
					emitterHTTPRouteContext.Spec.Rules[ruleIdx].Filters = append(emitterHTTPRouteContext.Spec.Rules[ruleIdx].Filters,
						gwapiv1.HTTPRouteFilter{
							Type: gwapiv1.HTTPRouteFilterURLRewrite,
							URLRewrite: &gwapiv1.HTTPURLRewriteFilter{
								Hostname: ptr.To(gwapiv1.PreciseHostname(val)),
							},
						},
					)

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressesWithHeaderMod[ingressNN] == nil {
						ingressesWithHeaderMod[ingressNN] = sets.New[string]()
					}
					ingressesWithHeaderMod[ingressNN].Insert("upstream-vhost")
				}
			}
		}

		// Notify once per Ingress if header modification was parsed
		for ingressNN, annotationSet := range ingressesWithHeaderMod {
			annotations := annotationSet.UnsortedList()
			notify(notifications.InfoNotification,
				fmt.Sprintf("parsed Header (%s) of ingress %s/%s",
					strings.Join(annotations, ", "), ingressNN.Namespace, ingressNN.Name),
				&emitterHTTPRouteContext.HTTPRoute)
		}

		eir.HTTPRoutes[key] = emitterHTTPRouteContext
	}

	if len(errList) > 0 {
		return errList
	}
	return nil
}
