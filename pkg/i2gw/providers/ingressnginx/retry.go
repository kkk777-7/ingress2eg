package ingressnginx

import (
	"fmt"
	"strconv"
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
	proxyNextUpstreamAnnotation      = "nginx.ingress.kubernetes.io/proxy-next-upstream"
	proxyNextUpstreamTriesAnnotation = "nginx.ingress.kubernetes.io/proxy-next-upstream-tries"
)

func retryFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

		// Track which Ingresses have retry and their annotation keys
		ingressesWithRetry := make(map[types.NamespacedName]sets.Set[string])

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

				if val := ingress.Annotations[proxyNextUpstreamAnnotation]; val != "" {
					var annotationKey, annotationMessage []string

					triggers := strings.Split(val, " ")
					for i, h := range triggers {
						triggers[i] = strings.TrimSpace(h)
					}

					retryIR := getOrCreateRetryIR(&emitterHTTPRouteContext, ruleIdx)
					retryIR.Triggers = triggers
					annotationKey = append(annotationKey, proxyNextUpstreamAnnotation)
					annotationMessage = append(annotationMessage, "proxy-next-upstream")

					if triesVal := ingress.Annotations[proxyNextUpstreamTriesAnnotation]; triesVal != "" {
						valInt, err := strconv.ParseInt(triesVal, 10, 32)
						if err != nil {
							notify(notifications.ErrorNotification, fmt.Sprintf("Failed to parse Retry tries for Ingress %s/%s: %v",
								ingress.Namespace, ingress.Name, err), &emitterHTTPRouteContext.HTTPRoute)
							continue
						}
						retryIR.Count = ptr.To(int32(valInt))
						annotationKey = append(annotationKey, proxyNextUpstreamTriesAnnotation)
						annotationMessage = append(annotationMessage, "proxy-next-upstream-tries")
					}

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: annotationKey,
					}
					retryIR.SetSource(extSource)

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressesWithRetry[ingressNN] == nil {
						ingressesWithRetry[ingressNN] = sets.New[string]()
					}
					ingressesWithRetry[ingressNN].Insert(annotationMessage...)
				}
			}
		}

		// Notify once per Ingress if retry was parsed
		for ingressNN, annotationSet := range ingressesWithRetry {
			annotations := annotationSet.UnsortedList()
			notify(notifications.InfoNotification,
				fmt.Sprintf("parsed Retry (%s) of ingress %s/%s",
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

func getOrCreateRetryIR(ctx *emitterir.HTTPRouteContext, ruleIdx int) *emitterir.RetryFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.RetryFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.RetryFeatureKey] = efMap
	}

	ef, exists := efMap[ruleIdx]
	if !exists {
		ef = &emitterir.RetryFeatureIR{}
		efMap[ruleIdx] = ef
	}
	return ef.(*emitterir.RetryFeatureIR)
}
