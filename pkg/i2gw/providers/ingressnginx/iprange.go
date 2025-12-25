package ingressnginx

import (
	"fmt"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"

	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
	providerir "github.com/kkk777-7/ingress2eg/pkg/i2gw/provider_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/providers/common"
)

const (
	whiteListSourceRangeAnnotation = "nginx.ingress.kubernetes.io/whitelist-source-range"
	denyListSourceRangeAnnotation  = "nginx.ingress.kubernetes.io/denylist-source-range"
)

func ipRangeFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

				if whiteList := ingress.Annotations[whiteListSourceRangeAnnotation]; whiteList != "" {
					ipRangeIR := getOrCreateIpRangeIR(&emitterHTTPRouteContext, ruleIdx)

					wl := strings.Split(whiteList, ",")
					for i, h := range wl {
						wl[i] = strings.TrimSpace(h)
					}
					ipRangeIR.AllowList = wl

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{whiteListSourceRangeAnnotation},
					}
					ipRangeIR.SetSource(extSource)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed IPRange (whitelist-source-range) annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
						&emitterHTTPRouteContext.HTTPRoute)
				}

				if denyList := ingress.Annotations[denyListSourceRangeAnnotation]; denyList != "" {
					ipRangeIR := getOrCreateIpRangeIR(&emitterHTTPRouteContext, ruleIdx)

					dl := strings.Split(denyList, ",")
					for i, h := range dl {
						dl[i] = strings.TrimSpace(h)
					}
					ipRangeIR.DenyList = dl

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{denyListSourceRangeAnnotation},
					}
					ipRangeIR.SetSource(extSource)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed IPRange (denylist-source-range) annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
						&emitterHTTPRouteContext.HTTPRoute)
				}
			}
		}
		eir.HTTPRoutes[key] = emitterHTTPRouteContext
	}

	if len(errList) > 0 {
		return errList
	}
	return nil
}

func getOrCreateIpRangeIR(ctx *emitterir.HTTPRouteContext, ruleIdx int) *emitterir.IpRangeFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.IpRangeFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.IpRangeFeatureKey] = efMap
	}

	ef, exists := efMap[ruleIdx]
	if !exists {
		ef = &emitterir.IpRangeFeatureIR{}
		efMap[ruleIdx] = ef
	}
	return ef.(*emitterir.IpRangeFeatureIR)
}
