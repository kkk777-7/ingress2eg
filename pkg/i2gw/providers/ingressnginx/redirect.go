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
	sslRedirectAnnotation       = "nginx.ingress.kubernetes.io/ssl-redirect"
	forceSslRedirectAnnotation  = "nginx.ingress.kubernetes.io/force-ssl-redirect"
	permanentRedirectAnnotation = "nginx.ingress.kubernetes.io/permanent-redirect"
	temporalRedirectAnnotation  = "nginx.ingress.kubernetes.io/temporal-redirect"
	appRootAnnotation           = "nginx.ingress.kubernetes.io/app-root"
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

		// Track which Ingresses have redirect and their annotation keys
		ingressesWithRedirect := make(map[types.NamespacedName]sets.Set[string])

		// Track app-root annotation for processing after all rules
		var hasAppRoot bool
		var appRootValue string
		var appRootIngress networkingv1.Ingress

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

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressesWithRedirect[ingressNN] == nil {
						ingressesWithRedirect[ingressNN] = sets.New[string]()
					}
					ingressesWithRedirect[ingressNN].Insert("ssl-redirect")
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

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressesWithRedirect[ingressNN] == nil {
						ingressesWithRedirect[ingressNN] = sets.New[string]()
					}
					ingressesWithRedirect[ingressNN].Insert("force-ssl-redirect")
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

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressesWithRedirect[ingressNN] == nil {
						ingressesWithRedirect[ingressNN] = sets.New[string]()
					}
					ingressesWithRedirect[ingressNN].Insert("permanent-redirect")
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

					ingressNN := types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name}
					if ingressesWithRedirect[ingressNN] == nil {
						ingressesWithRedirect[ingressNN] = sets.New[string]()
					}
					ingressesWithRedirect[ingressNN].Insert("temporal-redirect")
				}

				if val := ingress.Annotations[appRootAnnotation]; val != "" {
					if hasAppRoot && val != appRootValue {
						// Warn if multiple different app-root values are detected
						notify(notifications.WarningNotification,
							fmt.Sprintf("multiple app-root annotations detected: using %s from ingress %s/%s, ignoring %s from ingress %s/%s",
								appRootValue, appRootIngress.Namespace, appRootIngress.Name,
								val, ingress.Namespace, ingress.Name),
							&emitterHTTPRouteContext.HTTPRoute)
						continue
					}
					hasAppRoot = true
					appRootValue = val
					appRootIngress = ingress
				}
			}
		}

		// Process app-root annotation after all rules
		if hasAppRoot {
			processAppRootRedirect(&emitterHTTPRouteContext, appRootIngress, appRootValue, ingressesWithRedirect)
		}

		// Notify once per Ingress if redirect was parsed
		for ingressNN, annotationSet := range ingressesWithRedirect {
			annotations := annotationSet.UnsortedList()
			notify(notifications.InfoNotification,
				fmt.Sprintf("parsed Redirect (%s) of ingress %s/%s",
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

// processAppRootRedirect processes app-root annotation and creates/updates redirect rule
func processAppRootRedirect(
	ctx *emitterir.HTTPRouteContext,
	appRootIngress networkingv1.Ingress,
	appRootValue string,
	ingressesWithRedirect map[types.NamespacedName]sets.Set[string],
) {
	if len(ctx.Spec.Hostnames) == 0 {
		return
	}

	// Check if there's already a root exact match rule
	rootExactMatchRuleIdx := -1
	for idx, rule := range ctx.Spec.Rules {
		for _, match := range rule.Matches {
			if match.Path != nil && match.Path.Type != nil && *match.Path.Type == gwapiv1.PathMatchExact &&
				match.Path.Value != nil && *match.Path.Value == "/" {
				rootExactMatchRuleIdx = idx
				break
			}
		}
		if rootExactMatchRuleIdx != -1 {
			break
		}
	}

	targetRuleIdx := rootExactMatchRuleIdx
	if rootExactMatchRuleIdx == -1 {
		// Add a new rule for root exact match
		if len(ctx.Spec.Rules) > 0 {
			newRule := gwapiv1.HTTPRouteRule{
				Name: ptr.To(gwapiv1.SectionName("rule-for-app-root-redirect")),
				Matches: []gwapiv1.HTTPRouteMatch{
					{
						Path: &gwapiv1.HTTPPathMatch{
							Type:  ptr.To(gwapiv1.PathMatchExact),
							Value: ptr.To("/"),
						},
					},
				},
			}
			ctx.Spec.Rules = append(ctx.Spec.Rules, newRule)
			targetRuleIdx = len(ctx.Spec.Rules) - 1
		}
	}

	redirectIR := getOrCreateRedirectIR(ctx, targetRuleIdx)
	hostname := string(ctx.Spec.Hostnames[0])
	redirectIR.Url = constructRedirectURL(appRootIngress, hostname, appRootValue)
	redirectIR.StatusCode = 302

	source := &emitterir.ExtensionFeatureSource{
		IngressNN:     types.NamespacedName{Namespace: appRootIngress.Namespace, Name: appRootIngress.Name},
		AnnotationKey: []string{appRootAnnotation},
	}
	redirectIR.SetSource(source)

	ingressNN := types.NamespacedName{Namespace: appRootIngress.Namespace, Name: appRootIngress.Name}
	if ingressesWithRedirect[ingressNN] == nil {
		ingressesWithRedirect[ingressNN] = sets.New[string]()
	}
	ingressesWithRedirect[ingressNN].Insert("app-root")
}

// constructRedirectURL builds a redirect URL with the appropriate scheme (http/https)
// based on the Ingress TLS configuration.
func constructRedirectURL(ingress networkingv1.Ingress, hostname string, path string) string {
	if hostname == "" {
		return ""
	}

	// Determine scheme based on TLS configuration
	scheme := "http"
	if ingress.Spec.TLS != nil {
		for _, tls := range ingress.Spec.TLS {
			for _, host := range tls.Hosts {
				if host == hostname {
					scheme = "https"
					break
				}
			}
			if scheme == "https" {
				break
			}
		}
	}
	return fmt.Sprintf("%s://%s%s", scheme, hostname, path)
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
