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
	rewriteAnnotation = "nginx.ingress.kubernetes.io/rewrite-target"
	appRootAnnotation = "nginx.ingress.kubernetes.io/app-root"
)

func rewriteFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

				if val := ingress.Annotations[rewriteAnnotation]; val != "" {
					if emitterHTTPRouteContext.ExtensionFeatures == nil {
						emitterHTTPRouteContext.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
					}
					efMap, exists := emitterHTTPRouteContext.ExtensionFeatures[emitterir.RewriteFeatureKey]
					if !exists {
						efMap = make(map[int]emitterir.ExtensionFeatureIR)
						emitterHTTPRouteContext.ExtensionFeatures[emitterir.RewriteFeatureKey] = efMap
					}
					efMap[ruleIdx] = &emitterir.RewriteFeatureIR{
						Target: val,
					}

					source := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: rewriteAnnotation,
					}
					efMap[ruleIdx].SetSource(source)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed Rewrite annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
						&emitterHTTPRouteContext.HTTPRoute)
				}

				if val := ingress.Annotations[appRootAnnotation]; val != "" {
					if emitterHTTPRouteContext.ExtensionFeatures == nil {
						emitterHTTPRouteContext.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
					}
					efMap, exists := emitterHTTPRouteContext.ExtensionFeatures[emitterir.RewriteFeatureKey]
					if !exists {
						efMap = make(map[int]emitterir.ExtensionFeatureIR)
						emitterHTTPRouteContext.ExtensionFeatures[emitterir.RewriteFeatureKey] = efMap
					}
					efMap[ruleIdx] = &emitterir.RewriteFeatureIR{
						Target: val,
					}

					source := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: appRootAnnotation,
					}
					efMap[ruleIdx].SetSource(source)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed Rewrite (app-root) annotations of ingress %s/%s", ingress.Namespace, ingress.Name),
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
