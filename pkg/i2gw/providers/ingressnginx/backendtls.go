package ingressnginx

import (
	"fmt"
	"strconv"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	proxySslSecretAnnotation     = "nginx.ingress.kubernetes.io/proxy-ssl-secret"
	proxySslVerifyAnnotation     = "nginx.ingress.kubernetes.io/proxy-ssl-verify"
	proxySslNameAnnotation       = "nginx.ingress.kubernetes.io/proxy-ssl-name"
	proxySslServerNameAnnotation = "nginx.ingress.kubernetes.io/proxy-ssl-server-name"
)

var SecretGVK = schema.GroupVersionKind{
	Group:   "",
	Version: "v1",
	Kind:    "Secret",
}

func backendTLSFeature(ingresses []networkingv1.Ingress, _ map[types.NamespacedName]map[string]int32, pir *providerir.ProviderIR, eir *emitterir.EmitterIR) field.ErrorList {
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

				if val := ingress.Annotations[proxySslSecretAnnotation]; val != "" {
					var annotationKeys, annotationMessage []string

					secretNN := strings.Split(val, "/")
					backendTLSIR := getOrCreateBackendTlsIR(&emitterHTTPRouteContext, ruleIdx)
					backendTLSIR.CertificateRef = gwapiv1.LocalObjectReference{
						Group: gwapiv1.Group(SecretGVK.Group),
						Kind:  gwapiv1.Kind(SecretGVK.Kind),
						Name:  gwapiv1.ObjectName(secretNN[1]),
					}
					annotationKeys = append(annotationKeys, proxySslSecretAnnotation)
					annotationMessage = append(annotationMessage, "proxy-ssl-secret")

					if val := ingress.Annotations[proxySslVerifyAnnotation]; val != "" {
						valBool, err := strconv.ParseBool(val)
						if err != nil {
							notify(notifications.ErrorNotification, fmt.Sprintf("invalid proxy-ssl-verify annotation %q in ingress %s/%s: %v", val, ingress.Namespace, ingress.Name, err),
								&emitterHTTPRouteContext.HTTPRoute)
						} else {
							backendTLSIR.InsecureSkipVerify = ptr.To(!valBool)
							annotationKeys = append(annotationKeys, proxySslVerifyAnnotation)
							annotationMessage = append(annotationMessage, "proxy-ssl-verify")
						}
					}

					if val := ingress.Annotations[proxySslServerNameAnnotation]; val == "on" {
						if val := ingress.Annotations[proxySslNameAnnotation]; val != "" {
							backendTLSIR.Sni = val
							annotationKeys = append(annotationKeys, proxySslNameAnnotation)
							annotationKeys = append(annotationKeys, proxySslServerNameAnnotation)
							annotationMessage = append(annotationMessage, "proxy-ssl-name")
							annotationMessage = append(annotationMessage, "proxy-ssl-server-name")
						}
					}

					extSource := &emitterir.ExtensionFeatureSource{
						IngressNN:     types.NamespacedName{Namespace: ingress.Namespace, Name: ingress.Name},
						AnnotationKey: annotationKeys,
					}
					backendTLSIR.SetSource(extSource)

					notify(notifications.InfoNotification, fmt.Sprintf("parsed BackendTLS %v annotations of ingress %s/%s", annotationMessage, ingress.Namespace, ingress.Name),
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

func getOrCreateBackendTlsIR(ctx *emitterir.HTTPRouteContext, ruleIdx int) *emitterir.BackendTLSFeatureIR {
	if ctx.ExtensionFeatures == nil {
		ctx.ExtensionFeatures = make(map[emitterir.ExtensionFeatureKey]map[int]emitterir.ExtensionFeatureIR)
	}
	efMap, exists := ctx.ExtensionFeatures[emitterir.BackendTLSFeatureKey]
	if !exists {
		efMap = make(map[int]emitterir.ExtensionFeatureIR)
		ctx.ExtensionFeatures[emitterir.BackendTLSFeatureKey] = efMap
	}

	ef, exists := efMap[ruleIdx]
	if !exists {
		ef = &emitterir.BackendTLSFeatureIR{}
		efMap[ruleIdx] = ef
	}
	return ef.(*emitterir.BackendTLSFeatureIR)
}
