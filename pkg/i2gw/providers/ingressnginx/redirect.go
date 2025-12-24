package ingressnginx

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"

	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
	providerir "github.com/kkk777-7/ingress2eg/pkg/i2gw/provider_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/providers/common"
)

const (
	sslRedirectAnnotation       = "nginx.ingress.kubernetes.io/ssl-redirect"
	forceSslRedirectAnnotation  = "nginx.ingress.kubernetes.io/force-ssl-redirect"
	permanentRedirectAnnotation = "nginx.ingress.kubernetes.io/permanent-redirect"
	temporalRedirectAnnotation  = "nginx.ingress.kubernetes.io/temporal-redirect"
)

func redirectFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

				if val := ingress.Annotations[sslRedirectAnnotation]; val == "true" {
					redirectIR := getOrCreateRedirectIR(&emitterHTTPRouteContext, ruleIdx)
					redirectIR.Ssl = &emitterir.SslRefirectIR{
						Force: false,
					}

					source := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{sslRedirectAnnotation},
					}
					redirectIR.SetSource(source)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed Redirect (ssl) annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
						&emitterHTTPRouteContext.HTTPRoute)
				}

				if val := ingress.Annotations[forceSslRedirectAnnotation]; val == "true" {
					redirectIR := getOrCreateRedirectIR(&emitterHTTPRouteContext, ruleIdx)
					redirectIR.Ssl = &emitterir.SslRefirectIR{
						Force: true,
					}

					source := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{forceSslRedirectAnnotation},
					}
					redirectIR.SetSource(source)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed Redirect (force-ssl) annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
						&emitterHTTPRouteContext.HTTPRoute)
				}

				if val, ok := ingress.Annotations[permanentRedirectAnnotation]; ok {
					redirectIR := getOrCreateRedirectIR(&emitterHTTPRouteContext, ruleIdx)
					redirectIR.Url = val
					redirectIR.StatusCode = 301

					source := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{permanentRedirectAnnotation},
					}
					redirectIR.SetSource(source)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed Redirect (permanent) annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
						&emitterHTTPRouteContext.HTTPRoute)
				}

				if val, ok := ingress.Annotations[temporalRedirectAnnotation]; ok {
					redirectIR := getOrCreateRedirectIR(&emitterHTTPRouteContext, ruleIdx)
					redirectIR.Url = val
					redirectIR.StatusCode = 302

					source := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{temporalRedirectAnnotation},
					}
					redirectIR.SetSource(source)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed Redirect (temporal) annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
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

func getOrCreateRedirectIR(ctx *emitterir.HTTPRouteContext, ruleIdx int) *emitterir.RedirectFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.RedirectFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.RedirectFeatureKey] = efMap
	}

	ef, exists := efMap[ruleIdx]
	if !exists {
		ef = &emitterir.RedirectFeatureIR{}
		efMap[ruleIdx] = ef
	}
	return ef.(*emitterir.RedirectFeatureIR)
}
