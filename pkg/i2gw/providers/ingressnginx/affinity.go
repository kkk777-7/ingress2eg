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
	defaultCookieName = "INGRESSCOOKIE"

	affinityAnnotation              = "nginx.ingress.kubernetes.io/affinity"
	sessionCookieNameAnnotation     = "nginx.ingress.kubernetes.io/session-cookie-name"
	sessionCookieSamesiteAnnotation = "nginx.ingress.kubernetes.io/session-cookie-samesite"
	sessionCookieMaxAgeAnnotation   = "nginx.ingress.kubernetes.io/session-cookie-max-age"
	sessionCookieExpiresAnnotation  = "nginx.ingress.kubernetes.io/session-cookie-expires"
)

func affinityFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

		// Track which Ingresses have affinity and their annotation keys
		ingressesWithAffinity := make(map[types.NamespacedName]sets.Set[string])

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

				if val := ingress.Annotations[affinityAnnotation]; val == "cookie" {
					var annotationKey, annotationMessage []string
					affinityIR := getOrCreateAffinityIR(&emitterHTTPRouteContext, ruleIdx)
					annotationKey = append(annotationKey, affinityAnnotation)
					annotationMessage = append(annotationMessage, "affinity")

					if cookieName := ingress.Annotations[sessionCookieNameAnnotation]; cookieName != "" {
						affinityIR.CookieName = cookieName
						annotationKey = append(annotationKey, sessionCookieNameAnnotation)
						annotationMessage = append(annotationMessage, "session-cookie-name")
					} else {
						affinityIR.CookieName = defaultCookieName
					}

					if sameSite := ingress.Annotations[sessionCookieSamesiteAnnotation]; sameSite != "" {
						affinityIR.CookieSameSite = sameSite
						annotationKey = append(annotationKey, sessionCookieSamesiteAnnotation)
						annotationMessage = append(annotationMessage, "session-cookie-samesite")
					}

					if expire := ingress.Annotations[sessionCookieExpiresAnnotation]; expire != "" {
						affinityIR.CookieTTLSeconds = ptr.To(gwapiv1.Duration(expire) + "s")
						annotationKey = append(annotationKey, sessionCookieExpiresAnnotation)
						annotationMessage = append(annotationMessage, "session-cookie-expires")
					} else if maxAge := ingress.Annotations[sessionCookieMaxAgeAnnotation]; maxAge != "" {
						affinityIR.CookieTTLSeconds = ptr.To(gwapiv1.Duration(maxAge) + "s")
						annotationKey = append(annotationKey, sessionCookieMaxAgeAnnotation)
						annotationMessage = append(annotationMessage, "session-cookie-max-age")
					}

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: annotationKey,
					}
					affinityIR.SetSource(extSource)

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressesWithAffinity[ingressNN] == nil {
						ingressesWithAffinity[ingressNN] = sets.New[string]()
					}
					ingressesWithAffinity[ingressNN].Insert(annotationMessage...)
				}
			}
		}

		// Notify once per Ingress if affinity was parsed
		for ingressNN, annotationSet := range ingressesWithAffinity {
			annotations := annotationSet.UnsortedList()
			notify(notifications.InfoNotification,
				fmt.Sprintf("parsed Affinity (%s) of ingress %s/%s",
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

func getOrCreateAffinityIR(ctx *emitterir.HTTPRouteContext, ruleIdx int) *emitterir.AffinityFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.AffinityFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.AffinityFeatureKey] = efMap
	}

	ef, exists := efMap[ruleIdx]
	if !exists {
		ef = &emitterir.AffinityFeatureIR{}
		efMap[ruleIdx] = ef
	}
	return ef.(*emitterir.AffinityFeatureIR)
}
