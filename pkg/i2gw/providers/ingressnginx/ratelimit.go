package ingressnginx

import (
	"fmt"
	"strconv"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"

	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
	providerir "github.com/kkk777-7/ingress2eg/pkg/i2gw/provider_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/providers/common"
)

const (
	rateLimitPerSecondAnnotation = "nginx.ingress.kubernetes.io/limit-rps"
	rateLimitPerMinuteAnnotation = "nginx.ingress.kubernetes.io/limit-rpm"
	rateLimitWhiteListAnnotation = "nginx.ingress.kubernetes.io/limit-whitelist"
)

func rateLimitFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

				if val := ingress.Annotations[rateLimitPerMinuteAnnotation]; val != "" {
					valInt, err := strconv.Atoi(val)
					if err != nil || valInt < 0 {
						notify(notifications.ErrorNotification, fmt.Sprintf("invalid rate limit rpm annotation %q in ingress %s/%s", val, ingress.Namespace, ingress.Name),
							&emitterHTTPRouteContext.HTTPRoute)
						continue
					}

					rateLimitIR := getOrCreateRateLimitIR(&emitterHTTPRouteContext, ruleIdx)
					rateLimitIR.LimitValue = uint(valInt)
					rateLimitIR.Unit = emitterir.MinuteTimeUnit

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{rateLimitPerMinuteAnnotation},
					}
					rateLimitIR.SetSource(extSource)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed Ratelimit (limit-rpm) annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
						&emitterHTTPRouteContext.HTTPRoute)

					continue
				}

				if val := ingress.Annotations[rateLimitPerSecondAnnotation]; val != "" {
					valInt, err := strconv.Atoi(val)
					if err != nil || valInt < 0 {
						notify(notifications.ErrorNotification, fmt.Sprintf("invalid rate limit rps annotation %q in ingress %s/%s", val, ingress.Namespace, ingress.Name),
							&emitterHTTPRouteContext.HTTPRoute)
						continue
					}

					rateLimitIR := getOrCreateRateLimitIR(&emitterHTTPRouteContext, ruleIdx)
					rateLimitIR.LimitValue = uint(valInt)
					rateLimitIR.Unit = emitterir.SecondTimeUnit

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{rateLimitPerSecondAnnotation},
					}
					rateLimitIR.SetSource(extSource)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed Ratelimit (limit-rps) annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
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

func getOrCreateRateLimitIR(ctx *emitterir.HTTPRouteContext, ruleIdx int) *emitterir.RateLimitFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.RateLimitFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.RateLimitFeatureKey] = efMap
	}

	ef, exists := efMap[ruleIdx]
	if !exists {
		ef = &emitterir.RateLimitFeatureIR{}
		efMap[ruleIdx] = ef
	}
	return ef.(*emitterir.RateLimitFeatureIR)
}
