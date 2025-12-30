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
	// Only support basic
	authTypeAnnotation   = "nginx.ingress.kubernetes.io/auth-type"
	authSecretAnnotation = "nginx.ingress.kubernetes.io/auth-secret" // #nosec G101
	// Format: <namespace>/<secret-name>
	authTlsSecretAnnotation       = "nginx.ingress.kubernetes.io/auth-tls-secret" // #nosec G101
	authUrlAnnotation             = "nginx.ingress.kubernetes.io/auth-url"
	authResponseHeadersAnnotation = "nginx.ingress.kubernetes.io/auth-response-headers"
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

		// Track which Ingresses have which auth types
		ingressAuthTypes := make(map[types.NamespacedName]sets.Set[string])

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

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressAuthTypes[ingressNN] == nil {
						ingressAuthTypes[ingressNN] = sets.New[string]()
					}
					ingressAuthTypes[ingressNN].Insert("BasicAuth")
				}

				if val := ingress.Annotations[authTlsSecretAnnotation]; val != "" {
					gwKey := types.NamespacedName{
						Name:      string(emitterHTTPRouteContext.Spec.ParentRefs[0].Name),
						Namespace: ptr.Deref((*string)(emitterHTTPRouteContext.Spec.ParentRefs[0].Namespace), emitterHTTPRouteContext.Namespace),
					}
					emitterGatewayContext, exists := eir.Gateways[gwKey]
					if !exists {
						notify(notifications.ErrorNotification, fmt.Sprintf("cannot find Gateway %s referenced by HTTPRoute %s/%s for Auth TLS configuration",
							gwKey.String(), emitterHTTPRouteContext.Namespace, emitterHTTPRouteContext.Name),
							&emitterHTTPRouteContext.HTTPRoute)
						continue
					}

					// Find the HTTPS listeners that match the hostname from the rule group
					listenerIndices := findMatchingGatewayListenerIndex(&emitterGatewayContext, rg.Host, ptr.To("HTTPS"))
					if len(listenerIndices) == 0 {
						notify(notifications.ErrorNotification, fmt.Sprintf("cannot find matching HTTPS listener for hostname %q in Gateway %s for mTLS configuration",
							rg.Host, gwKey.String()),
							&emitterHTTPRouteContext.HTTPRoute)
						continue
					}

					nn := strings.Split(val, "/")
					for _, listenerIdx := range listenerIndices {
						mtlsIR := getOrCreateMTLSAuthIR(&emitterGatewayContext, listenerIdx)
						mtlsIR.Name = nn[1]
						mtlsIR.Namespace = nn[0]

						extSource := &emitterir.ExtensionFeatureSource{
							IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
							AnnotationKey: []string{authTlsSecretAnnotation},
						}
						mtlsIR.SetSource(extSource)
					}

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressAuthTypes[ingressNN] == nil {
						ingressAuthTypes[ingressNN] = sets.New[string]()
					}
					ingressAuthTypes[ingressNN].Insert("mTLS")

					eir.Gateways[gwKey] = emitterGatewayContext
				}

				if val := ingress.Annotations[authUrlAnnotation]; val != "" {
					externalAuthIR := getOrCreateExternalAuthIR(&emitterHTTPRouteContext, ruleIdx)
					externalAuthIR.Url = val

					if headers := ingress.Annotations[authResponseHeadersAnnotation]; headers != "" {
						headerList := strings.Split(headers, ",")
						for i, h := range headerList {
							headerList[i] = strings.TrimSpace(h)
						}
						externalAuthIR.AllowedResponseHeaders = headerList
					}

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{authUrlAnnotation, authResponseHeadersAnnotation},
					}
					externalAuthIR.SetSource(extSource)

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressAuthTypes[ingressNN] == nil {
						ingressAuthTypes[ingressNN] = sets.New[string]()
					}
					ingressAuthTypes[ingressNN].Insert("ExternalAuth")
				}
			}
		}

		// Notify once per Ingress if any auth features were parsed
		for ingressNN, authTypeSet := range ingressAuthTypes {
			authTypes := authTypeSet.UnsortedList()
			notify(notifications.InfoNotification,
				fmt.Sprintf("parsed Auth (%s) of ingress %s/%s",
					strings.Join(authTypes, ", "), ingressNN.Namespace, ingressNN.Name),
				&emitterHTTPRouteContext.HTTPRoute)
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

func getOrCreateMTLSAuthIR(ctx *emitterir.GatewayContext, listenerIdx int) *emitterir.MTLSFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.MTLSFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.MTLSFeatureKey] = efMap
	}

	ef, exists := efMap[listenerIdx]
	if !exists {
		ef = &emitterir.MTLSFeatureIR{}
		efMap[listenerIdx] = ef
	}
	return ef.(*emitterir.MTLSFeatureIR)
}

func getOrCreateExternalAuthIR(ctx *emitterir.HTTPRouteContext, ruleIdx int) *emitterir.ExternalAuthFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.ExternalAuthFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.ExternalAuthFeatureKey] = efMap
	}

	ef, exists := efMap[ruleIdx]
	if !exists {
		ef = &emitterir.ExternalAuthFeatureIR{}
		efMap[ruleIdx] = ef
	}
	return ef.(*emitterir.ExternalAuthFeatureIR)
}

// findMatchingGatewayListenerIndex finds a Gateway HTTPS listener index that matches the given hostname.
// This is used to apply mTLS configuration to the specific listener handling the hostname from a backendSource.
// Only HTTPS protocol listeners are considered since mTLS requires TLS.
//
// Matching logic:
// - Only matches listeners with protocol HTTPS
// - If hostname is empty (""), matches the first HTTPS listener with nil hostname (wildcard listener)
// - If hostname is specified, matches the first HTTPS listener with exact hostname match
//
// Returns the listener index, or -1 if no match is found.
func findMatchingGatewayListenerIndex(gateway *emitterir.GatewayContext, hostname string, specifyProtocol *string) []int {
	var matchingIndices []int

	for listenerIdx, listener := range gateway.Spec.Listeners {
		if specifyProtocol != nil && listener.Protocol != gwapiv1.ProtocolType(*specifyProtocol) {
			continue
		}

		// If hostname is empty, match wildcard listeners (hostname == nil)
		if hostname == "" {
			if listener.Hostname == nil {
				matchingIndices = append(matchingIndices, listenerIdx)
			}
			continue
		}

		// If hostname is specified, match exact hostname
		if listener.Hostname != nil && string(*listener.Hostname) == hostname {
			matchingIndices = append(matchingIndices, listenerIdx)
		}
	}

	return matchingIndices
}
