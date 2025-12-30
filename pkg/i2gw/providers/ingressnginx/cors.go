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
	enableCorsAnnotation           = "nginx.ingress.kubernetes.io/enable-cors"
	corsAllowOriginAnnotation      = "nginx.ingress.kubernetes.io/cors-allow-origin"
	corsAllowMethodsAnnotation     = "nginx.ingress.kubernetes.io/cors-allow-methods"
	corsAllowHeadersAnnotation     = "nginx.ingress.kubernetes.io/cors-allow-headers"
	corsExposeHeadersAnnotation    = "nginx.ingress.kubernetes.io/cors-expose-headers"
	corsAllowCredentialsAnnotation = "nginx.ingress.kubernetes.io/cors-allow-credentials" // #nosec G101
	corsMaxAgeAnnotation           = "nginx.ingress.kubernetes.io/cors-max-age"
)

func corsFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

		// Track which Ingresses have CORS and their annotation keys
		ingressesWithCORS := make(map[types.NamespacedName]sets.Set[string])

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

				if val := ingress.Annotations[enableCorsAnnotation]; val == "true" {
					var annotationKeys, annotationMessage []string
					corsIR := getOrCreateCorsIR(&emitterHTTPRouteContext, ruleIdx)
					annotationKeys = append(annotationKeys, enableCorsAnnotation)
					annotationMessage = append(annotationMessage, "enable-cors")

					if val := ingress.Annotations[corsAllowOriginAnnotation]; val != "" {
						origins := strings.Split(val, ",")
						for i, h := range origins {
							origins[i] = strings.TrimSpace(h)
						}
						corsIR.AllowOrigins = origins
						annotationKeys = append(annotationKeys, corsAllowOriginAnnotation)
						annotationMessage = append(annotationMessage, "cors-allow-origin")
					}
					if val := ingress.Annotations[corsAllowMethodsAnnotation]; val != "" {
						methods := strings.Split(val, ",")
						for i, h := range methods {
							methods[i] = strings.TrimSpace(h)
						}
						corsIR.AllowMethods = methods
						annotationKeys = append(annotationKeys, corsAllowMethodsAnnotation)
						annotationMessage = append(annotationMessage, "cors-allow-methods")
					}
					if val := ingress.Annotations[corsAllowHeadersAnnotation]; val != "" {
						headers := strings.Split(val, ",")
						for i, h := range headers {
							headers[i] = strings.TrimSpace(h)
						}
						corsIR.AllowHeaders = headers
						annotationKeys = append(annotationKeys, corsAllowHeadersAnnotation)
						annotationMessage = append(annotationMessage, "cors-allow-headers")
					}
					if val := ingress.Annotations[corsExposeHeadersAnnotation]; val != "" {
						exposeHeaders := strings.Split(val, ",")
						for i, h := range exposeHeaders {
							exposeHeaders[i] = strings.TrimSpace(h)
						}
						corsIR.ExposeHeaders = exposeHeaders
						annotationKeys = append(annotationKeys, corsExposeHeadersAnnotation)
						annotationMessage = append(annotationMessage, "cors-expose-headers")
					}
					if val := ingress.Annotations[corsAllowCredentialsAnnotation]; val != "" {
						if val == "true" {
							corsIR.AllowCredentials = ptr.To(true)
						} else {
							corsIR.AllowCredentials = ptr.To(false)
						}
						annotationKeys = append(annotationKeys, corsAllowCredentialsAnnotation)
						annotationMessage = append(annotationMessage, "cors-allow-credentials")
					}
					if val := ingress.Annotations[corsMaxAgeAnnotation]; val != "" {
						corsIR.MaxAge = ptr.To(gwapiv1.Duration(val) + "s")
						annotationKeys = append(annotationKeys, corsMaxAgeAnnotation)
						annotationMessage = append(annotationMessage, "cors-max-age")
					}

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: annotationKeys,
					}
					corsIR.SetSource(extSource)

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressesWithCORS[ingressNN] == nil {
						ingressesWithCORS[ingressNN] = sets.New[string]()
					}
					ingressesWithCORS[ingressNN].Insert(annotationMessage...)
				}
			}
		}

		// Notify once per Ingress if CORS was parsed
		for ingressNN, annotationSet := range ingressesWithCORS {
			annotations := annotationSet.UnsortedList()
			notify(notifications.InfoNotification,
				fmt.Sprintf("parsed CORS (%s) of ingress %s/%s",
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

func getOrCreateCorsIR(ctx *emitterir.HTTPRouteContext, ruleIdx int) *emitterir.CORSFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.CORSFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.CORSFeatureKey] = efMap
	}

	ef, exists := efMap[ruleIdx]
	if !exists {
		ef = &emitterir.CORSFeatureIR{}
		efMap[ruleIdx] = ef
	}
	return ef.(*emitterir.CORSFeatureIR)
}
