package ingressnginx

import (
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwapiv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
	providerir "github.com/kkk777-7/ingress2eg/pkg/i2gw/provider_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/providers/common"
)

const (
	sslPassThroughAnnotation = "nginx.ingress.kubernetes.io/ssl-passthrough" // #nosec G101
)

func sslPassthroughFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

		var hasSslPassthrough bool
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

				if val := ingress.Annotations[sslPassThroughAnnotation]; val == "true" {
					hasSslPassthrough = true
					break
				}
			}
		}

		if hasSslPassthrough {
			gwNN := types.NamespacedName{
				Name:      string(emitterHTTPRouteContext.Spec.ParentRefs[0].Name),
				Namespace: ptr.Deref((*string)(emitterHTTPRouteContext.Spec.ParentRefs[0].Namespace), emitterHTTPRouteContext.Namespace),
			}
			var hostname *gwapiv1.Hostname
			if len(emitterHTTPRouteContext.Spec.Hostnames) > 0 {
				hostname = &emitterHTTPRouteContext.Spec.Hostnames[0]
			}
			listenerName := ensureTLSListenerForPassthrough(eir, gwNN, hostname)

			tlsRouteContext := convertHTTPRulesToTLSRoute(emitterHTTPRouteContext, listenerName)

			tlsKey := types.NamespacedName{
				Namespace: key.Namespace,
				Name:      key.Name,
			}
			eir.TLSRoutes[tlsKey] = tlsRouteContext

			notify(notifications.InfoNotification, "created TLSRoute from HTTPRoute with SSL Passthrough enabled",
				&tlsRouteContext.TLSRoute)

			// Remove the original HTTPRoute
			delete(eir.HTTPRoutes, key)
		}
	}

	if len(errList) > 0 {
		return errList
	}
	return nil
}

func convertHTTPRulesToTLSRoute(httpRouteCtx emitterir.HTTPRouteContext, sectionName gwapiv1.SectionName) emitterir.TLSRouteContext {
	tlsRoute := gwapiv1a2.TLSRoute{
		ObjectMeta: httpRouteCtx.ObjectMeta,
		Spec: gwapiv1a2.TLSRouteSpec{
			CommonRouteSpec: httpRouteCtx.Spec.CommonRouteSpec,
			Hostnames:       httpRouteCtx.Spec.Hostnames,
			Rules:           make([]gwapiv1a2.TLSRouteRule, len(httpRouteCtx.Spec.Rules)),
		},
	}
	if sectionName != "" {
		// ParentRefs should have one entry
		tlsRoute.Spec.ParentRefs[0].SectionName = ptr.To(sectionName)
	}

	for i, httpRule := range httpRouteCtx.Spec.Rules {
		tlsRule := gwapiv1a2.TLSRouteRule{
			BackendRefs: convertHTTPBackendRefToTLSBackendRef(httpRule.BackendRefs),
		}
		if httpRule.Name != nil {
			tlsRule.Name = httpRule.Name
		}

		tlsRoute.Spec.Rules[i] = tlsRule
	}
	tlsRoute.SetGroupVersionKind(common.TLSRouteGVK)

	return emitterir.TLSRouteContext{
		TLSRoute: tlsRoute,
	}
}

func convertHTTPBackendRefToTLSBackendRef(httpBackendRefs []gwapiv1.HTTPBackendRef) []gwapiv1a2.BackendRef {
	tlsBackendRefs := make([]gwapiv1a2.BackendRef, len(httpBackendRefs))

	for i, httpRef := range httpBackendRefs {
		tlsBackendRefs[i] = gwapiv1a2.BackendRef{
			BackendObjectReference: httpRef.BackendObjectReference,
		}
	}

	return tlsBackendRefs
}

// ensureTLSListenerForPassthrough replaces any existing HTTPS listener with a TLS Passthrough listener
// for the given hostname, or adds a new one if none exists. Returns the listener name.
func ensureTLSListenerForPassthrough(eir *emitterir.EmitterIR, gwNN types.NamespacedName, hostname *gwapiv1.Hostname) gwapiv1.SectionName {
	gw, ok := eir.Gateways[gwNN]
	if !ok {
		// should not happen
		return ""
	}

	listenerName := "tls"
	if hostname != nil {
		listenerName = fmt.Sprintf("%s-tls", common.NameFromHost(string(*hostname)))
	}

	// Filter out HTTPS listeners with matching hostname and check for existing TLS listener
	var newListeners []gwapiv1.Listener
	for _, listener := range gw.Spec.Listeners {
		hostnameMatches := (hostname == nil) == (listener.Hostname == nil) &&
			(hostname == nil || *hostname == *listener.Hostname)

		// Skip HTTPS listeners with matching hostname (will be replaced with TLS Passthrough)
		if listener.Protocol == gwapiv1.HTTPSProtocolType && hostnameMatches {
			continue
		}

		// If TLS listener already exists with matching hostname, return its name
		if listener.Protocol == gwapiv1.TLSProtocolType && hostnameMatches {
			return listener.Name
		}

		newListeners = append(newListeners, listener)
	}

	// Add TLS Passthrough listener
	newListener := gwapiv1.Listener{
		Name:     gwapiv1.SectionName(listenerName),
		Protocol: gwapiv1.TLSProtocolType,
		Port:     gwapiv1.PortNumber(443),
		TLS: &gwapiv1.ListenerTLSConfig{
			Mode: ptr.To(gwapiv1.TLSModePassthrough),
		},
		AllowedRoutes: &gwapiv1.AllowedRoutes{
			Namespaces: &gwapiv1.RouteNamespaces{
				From: ptr.To(gwapiv1.NamespacesFromSame),
			},
			Kinds: []gwapiv1.RouteGroupKind{
				{
					Group: ptr.To(gwapiv1.Group(common.TLSRouteGVK.Group)),
					Kind:  gwapiv1.Kind(common.TLSRouteGVK.Kind),
				},
			},
		},
	}
	if hostname != nil {
		newListener.Hostname = hostname
	}
	newListeners = append(newListeners, newListener)

	gw.Spec.Listeners = newListeners
	eir.Gateways[gwNN] = gw

	return gwapiv1.SectionName(listenerName)
}
