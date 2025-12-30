package ingressnginx

import (
	"fmt"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"

	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
	providerir "github.com/kkk777-7/ingress2eg/pkg/i2gw/provider_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/providers/common"
)

const (
	proxyBodySizeAnnotation = "nginx.ingress.kubernetes.io/proxy-body-size"
)

func proxyBufferFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

		// Track which Ingresses have buffer configuration and their annotation keys
		ingressesWithBuffer := make(map[types.NamespacedName]sets.Set[string])

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

				if val := ingress.Annotations[proxyBodySizeAnnotation]; val != "" {
					gwKey := types.NamespacedName{
						Name:      string(emitterHTTPRouteContext.Spec.ParentRefs[0].Name),
						Namespace: ptr.Deref((*string)(emitterHTTPRouteContext.Spec.ParentRefs[0].Namespace), emitterHTTPRouteContext.Namespace),
					}
					emitterGatewayContext, exists := eir.Gateways[gwKey]
					if !exists {
						notify(notifications.ErrorNotification, fmt.Sprintf("cannot find Gateway %s referenced by HTTPRoute %s/%s for Buffer configuration",
							gwKey.String(), emitterHTTPRouteContext.Namespace, emitterHTTPRouteContext.Name),
							&emitterHTTPRouteContext.HTTPRoute)
						continue
					}

					// Find the listeners that match the hostname from the rule group
					listenerIndices := findMatchingGatewayListenerIndex(&emitterGatewayContext, rg.Host, nil)
					if len(listenerIndices) == 0 {
						notify(notifications.ErrorNotification, fmt.Sprintf("cannot find matching listener for hostname %q in Gateway %s for Buffer configuration",
							rg.Host, gwKey.String()),
							&emitterHTTPRouteContext.HTTPRoute)
						continue
					}

					for _, listenerIdx := range listenerIndices {
						bufferIR := getOrCreateGatewayBufferIR(&emitterGatewayContext, listenerIdx)
						bufferIR.LimitValue = val

						extSource := &emitterir.ExtensionFeatureSource{
							IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
							AnnotationKey: []string{proxyBodySizeAnnotation},
						}
						bufferIR.SetSource(extSource)
					}

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressesWithBuffer[ingressNN] == nil {
						ingressesWithBuffer[ingressNN] = sets.New[string]()
					}
					ingressesWithBuffer[ingressNN].Insert("proxy-body-size")

					eir.Gateways[gwKey] = emitterGatewayContext
				}
			}
		}

		// Notify once per Ingress if buffer configuration was parsed
		for ingressNN, annotationSet := range ingressesWithBuffer {
			annotations := annotationSet.UnsortedList()
			notify(notifications.InfoNotification,
				fmt.Sprintf("parsed Buffer (%s) of ingress %s/%s",
					strings.Join(annotations, ", "), ingressNN.Namespace, ingressNN.Name),
				&emitterHTTPRouteContext.HTTPRoute)
		}
	}

	if len(errList) > 0 {
		return errList
	}
	return nil
}

func getOrCreateGatewayBufferIR(ctx *emitterir.GatewayContext, listenerIdx int) *emitterir.BufferFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.BufferFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.BufferFeatureKey] = efMap
	}

	ef, exists := efMap[listenerIdx]
	if !exists {
		ef = &emitterir.BufferFeatureIR{}
		efMap[listenerIdx] = ef
	}
	return ef.(*emitterir.BufferFeatureIR)
}
