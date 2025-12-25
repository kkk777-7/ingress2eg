package ingressnginx

import (
	"fmt"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
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
	authTlsSecretAnnotation = "nginx.ingress.kubernetes.io/auth-tls-secret"
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

				if val := ingress.Annotations[authTlsSecretAnnotation]; val != "" {
					gwKey := types.NamespacedName{
						Name:      string(emitterHTTPRouteContext.Spec.ParentRefs[0].Name),
						Namespace: ptr.Deref((*string)(emitterHTTPRouteContext.Spec.ParentRefs[0].Namespace), emitterHTTPRouteContext.HTTPRoute.Namespace),
					}
					emitterGatewayContext, exists := eir.Gateways[gwKey]
					if !exists {
						notify(notifications.ErrorNotification, fmt.Sprintf("cannot find Gateway %s referenced by HTTPRoute %s/%s for Auth TLS configuration",
							gwKey.String(), emitterHTTPRouteContext.HTTPRoute.Namespace, emitterHTTPRouteContext.HTTPRoute.Name),
							&emitterHTTPRouteContext.HTTPRoute)
						continue
					}

					// Find the HTTPS listener that matches the hostname from the rule group
					listenerIdx := findMatchingGatewayListenerIndex(&emitterGatewayContext, rg.Host)
					if listenerIdx == -1 {
						notify(notifications.ErrorNotification, fmt.Sprintf("cannot find matching HTTPS listener for hostname %q in Gateway %s for mTLS configuration",
							rg.Host, gwKey.String()),
							&emitterHTTPRouteContext.HTTPRoute)
						continue
					}

					nn := strings.Split(val, "/")
					mtlsIR := getOrCreateMTLSAuthIR(&emitterGatewayContext, listenerIdx)
					mtlsIR.Name = nn[1]
					mtlsIR.Namespace = nn[0]

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: []string{authTlsSecretAnnotation},
					}
					mtlsIR.SetSource(extSource)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed Auth (tls-secret) annotation of ingress %s/%s", ingress.Namespace, ingress.Name),
						&emitterHTTPRouteContext.HTTPRoute)

					eir.Gateways[gwKey] = emitterGatewayContext
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
func findMatchingGatewayListenerIndex(gateway *emitterir.GatewayContext, hostname string) int {
	for listenerIdx, listener := range gateway.Spec.Listeners {
		// Only consider HTTPS listeners for mTLS
		if listener.Protocol != gwapiv1.HTTPSProtocolType {
			continue
		}

		// If hostname is empty, match wildcard listeners (hostname == nil)
		if hostname == "" {
			if listener.Hostname == nil {
				return listenerIdx
			}
			continue
		}

		// If hostname is specified, match exact hostname
		if listener.Hostname != nil && string(*listener.Hostname) == hostname {
			return listenerIdx
		}
	}

	// No matching listener found
	return -1
}
