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
	// Only support basic
	authTypeAnnotation   = "nginx.ingress.kubernetes.io/auth-type"
	authSecretAnnotation = "nginx.ingress.kubernetes.io/auth-secret"
)

func authFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

				if val := ingress.Annotations[authTypeAnnotation]; val != "" {
					if val != "basic" {
						notify(notifications.ErrorNotification, fmt.Sprintf("unsupported auth-type %q in Ingress %s/%s, only 'basic' is supported",
							val, ingress.Namespace, ingress.Name), &emitterHTTPRouteContext.HTTPRoute)
						continue
					}

					secretName := ingress.Annotations[authSecretAnnotation]
					if secretName == "" {
						notify(notifications.ErrorNotification, fmt.Sprintf("missing auth-secret annotation in Ingress %s/%s",
							ingress.Namespace, ingress.Name), &emitterHTTPRouteContext.HTTPRoute)
						continue
					}

					basicAuthIR := getOrCreateBasicAuthIR(&emitterHTTPRouteContext, ruleIdx)
					basicAuthIR.Name = secretName
					basicAuthIR.Namespace = ingress.Namespace

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{authTypeAnnotation, authSecretAnnotation},
					}
					basicAuthIR.SetSource(extSource)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed Auth (auth-type, auth-secret) annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
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

func getOrCreateBasicAuthIR(ctx *emitterir.HTTPRouteContext, ruleIdx int) *emitterir.BasicAuthFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.BasicAuthFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.BasicAuthFeatureKey] = efMap
	}

	ef, exists := efMap[ruleIdx]
	if !exists {
		ef = &emitterir.BasicAuthFeatureIR{}
		efMap[ruleIdx] = ef
	}
	return ef.(*emitterir.BasicAuthFeatureIR)
}
