package ingressnginx

import (
	"fmt"

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
	backendProtocolAnnotation = "nginx.ingress.kubernetes.io/backend-protocol"
)

func backendProtocolFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

		grpcRules := []gwapiv1.HTTPRouteRule{}
		httpRules := []gwapiv1.HTTPRouteRule{}
		grpcRuleOriginalIndices := sets.New[int]()
		httpRuleOriginalIndices := sets.New[int]()

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
				if val := ingress.Annotations[backendProtocolAnnotation]; val != "" {
					switch val {
					case "GRPC", "GRPCS":
						grpcRules = append(grpcRules, emitterHTTPRouteContext.Spec.Rules[ruleIdx])
						grpcRuleOriginalIndices.Insert(ruleIdx)
					case "HTTP", "HTTPS":
						httpRules = append(httpRules, emitterHTTPRouteContext.Spec.Rules[ruleIdx])
						httpRuleOriginalIndices.Insert(ruleIdx)
					default:
						notify(notifications.WarningNotification, fmt.Sprintf("backend-protocol %q is not supported, treating as HTTP for ingress rule %d",
							val, ruleIdx), &emitterHTTPRouteContext.HTTPRoute)
						// Fall back to HTTP
						httpRules = append(httpRules, emitterHTTPRouteContext.Spec.Rules[ruleIdx])
						httpRuleOriginalIndices.Insert(ruleIdx)
					}
				} else {
					// Default to HTTP
					httpRules = append(httpRules, emitterHTTPRouteContext.Spec.Rules[ruleIdx])
					httpRuleOriginalIndices.Insert(ruleIdx)
				}
			}
		}

		// If no GRPC rules, continue to next route
		if len(grpcRules) == 0 {
			continue
		}

		// Create GRPCRoute from GRPC rules
		if len(grpcRules) > 0 {
			grpcRouteContext := convertHTTPRulesToGRPCRoute(emitterHTTPRouteContext, grpcRules)

			grpcRuleList := grpcRuleOriginalIndices.UnsortedList()
			if backendTLSFeatures, ok := emitterHTTPRouteContext.ExtensionFeatures[emitterir.BackendTLSFeatureKey]; ok {
				if grpcRouteContext.ExtensionFeatures == nil {
					grpcRouteContext.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
				}
				// Copy only the BackendTLS features for GRPC rules
				grpcBackendTLSFeatures := make(map[int]emitterir.ExtensionFeatureIR)
				for newIdx, originalIdx := range grpcRuleList {
					if feature, ok := backendTLSFeatures[originalIdx]; ok {
						grpcBackendTLSFeatures[newIdx] = feature
					}
				}
				if len(grpcBackendTLSFeatures) > 0 {
					grpcRouteContext.ExtensionFeatures[emitterir.BackendTLSFeatureKey] = grpcBackendTLSFeatures
				}
			}

			grpcKey := types.NamespacedName{
				Namespace: key.Namespace,
				Name:      key.Name,
			}
			eir.GRPCRoutes[grpcKey] = grpcRouteContext
			notify(notifications.InfoNotification, fmt.Sprintf("created GRPCRoute with %d rule(s) from HTTPRoute", len(grpcRules)),
				&grpcRouteContext.GRPCRoute)
		}

		// Update or remove HTTPRoute
		if len(httpRules) > 0 {
			// Keep HTTPRoute with only HTTP rules
			emitterHTTPRouteContext.Spec.Rules = httpRules

			httpRuleList := httpRuleOriginalIndices.UnsortedList()
			if backendTLSFeatures, ok := emitterHTTPRouteContext.ExtensionFeatures[emitterir.BackendTLSFeatureKey]; ok {
				httpBackendTLSFeatures := make(map[int]emitterir.ExtensionFeatureIR)
				for newIdx, originalIdx := range httpRuleList {
					if feature, ok := backendTLSFeatures[originalIdx]; ok {
						httpBackendTLSFeatures[newIdx] = feature
					}
				}
				if len(httpBackendTLSFeatures) > 0 {
					emitterHTTPRouteContext.ExtensionFeatures[emitterir.BackendTLSFeatureKey] = httpBackendTLSFeatures
				} else {
					delete(emitterHTTPRouteContext.ExtensionFeatures, emitterir.BackendTLSFeatureKey)
				}
			}
			eir.HTTPRoutes[key] = emitterHTTPRouteContext
		} else {
			// All rules were GRPC, remove HTTPRoute
			delete(eir.HTTPRoutes, key)
		}
	}

	if len(errList) > 0 {
		return errList
	}
	return nil
}

func convertHTTPRulesToGRPCRoute(httpRouteCtx emitterir.HTTPRouteContext, httpRules []gwapiv1.HTTPRouteRule) emitterir.GRPCRouteContext {
	grpcRoute := gwapiv1.GRPCRoute{
		ObjectMeta: httpRouteCtx.ObjectMeta,
		Spec: gwapiv1.GRPCRouteSpec{
			CommonRouteSpec: httpRouteCtx.Spec.CommonRouteSpec,
			Hostnames:       httpRouteCtx.Spec.Hostnames,
			Rules:           make([]gwapiv1.GRPCRouteRule, len(httpRules)),
		},
	}

	for i, httpRule := range httpRules {
		grpcRule := gwapiv1.GRPCRouteRule{
			Matches:     convertHTTPMatchToGRPCMatch(httpRule.Matches),
			BackendRefs: convertHTTPBackendRefToGRPCBackendRef(httpRule.BackendRefs),
		}
		if httpRule.Name != nil {
			grpcRule.Name = httpRule.Name
		}

		if len(httpRule.Filters) > 0 {
			filterResult := common.ConvertHTTPFiltersToGRPCFilters(httpRule.Filters)
			grpcRule.Filters = filterResult.GRPCFilters

			// Log unsupported filters
			for _, unsupportedType := range filterResult.UnsupportedTypes {
				notify(notifications.WarningNotification, fmt.Sprintf("HTTPRoute filter type %s is not supported in GRPCRoute", unsupportedType), &grpcRoute)
			}
		}

		grpcRoute.Spec.Rules[i] = grpcRule
	}
	grpcRoute.SetGroupVersionKind(common.GRPCRouteGVK)

	return emitterir.GRPCRouteContext{
		GRPCRoute: grpcRoute,
	}
}

func convertHTTPMatchToGRPCMatch(httpMatches []gwapiv1.HTTPRouteMatch) []gwapiv1.GRPCRouteMatch {
	if len(httpMatches) == 0 {
		return nil
	}

	grpcMatches := make([]gwapiv1.GRPCRouteMatch, 0, len(httpMatches))

	for _, httpMatch := range httpMatches {
		grpcMatch := gwapiv1.GRPCRouteMatch{}

		// Convert Path to Method using ParseGRPCServiceMethod
		if httpMatch.Path != nil && httpMatch.Path.Value != nil {
			service, method := common.ParseGRPCServiceMethod(*httpMatch.Path.Value)

			var methodMatchType *gwapiv1.GRPCMethodMatchType
			if httpMatch.Path.Type != nil {
				methodMatchType = ptr.To(gwapiv1.GRPCMethodMatchType(*httpMatch.Path.Type))
			}

			grpcMatch.Method = &gwapiv1.GRPCMethodMatch{
				Type: methodMatchType,
			}

			if service != "" {
				grpcMatch.Method.Service = ptr.To(service)
			}
			if method != "" {
				grpcMatch.Method.Method = ptr.To(method)
			}
		}

		// Convert Headers
		if len(httpMatch.Headers) > 0 {
			grpcMatch.Headers = make([]gwapiv1.GRPCHeaderMatch, len(httpMatch.Headers))
			for i, h := range httpMatch.Headers {
				var headerMatchType *gwapiv1.GRPCHeaderMatchType
				if h.Type != nil {
					headerMatchType = ptr.To(gwapiv1.GRPCHeaderMatchType(*h.Type))
				}
				grpcMatch.Headers[i] = gwapiv1.GRPCHeaderMatch{
					Type:  headerMatchType,
					Name:  gwapiv1.GRPCHeaderName(h.Name),
					Value: h.Value,
				}
			}
		}

		grpcMatches = append(grpcMatches, grpcMatch)
	}

	return grpcMatches
}

func convertHTTPBackendRefToGRPCBackendRef(httpBackendRefs []gwapiv1.HTTPBackendRef) []gwapiv1.GRPCBackendRef {
	grpcBackendRefs := make([]gwapiv1.GRPCBackendRef, len(httpBackendRefs))

	for i, httpRef := range httpBackendRefs {
		grpcBackendRefs[i] = gwapiv1.GRPCBackendRef{
			BackendRef: httpRef.BackendRef,
		}
	}

	return grpcBackendRefs
}
