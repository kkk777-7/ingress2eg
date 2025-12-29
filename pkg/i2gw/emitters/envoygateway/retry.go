package envoygateway_emitter

import (
	"fmt"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

func (e *Emitter) EmitRetry(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		ctx.MergeExtensionFeature(emitterir.RetryFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.RetryFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			retryIR := ir.(*emitterir.RetryFeatureIR)

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.RouteRuleAllIndex && idx < len(ctx.Spec.Rules) {
				sectionName = ctx.Spec.Rules[idx].Name
			}
			backendTrafficPolicy := e.getOrBuildBackendTrafficPolicy(ctx, sectionName, idx)
			if backendTrafficPolicy.Spec.Retry == nil {
				backendTrafficPolicy.Spec.Retry = &egapiv1a1.Retry{}
			}

			triggers, statusCodes, err := e.convertRetryTriggersToEnvoy(retryIR.Triggers)
			if err != nil {
				notify(notifications.ErrorNotification, fmt.Sprintf("failed to convert Retry annotations for ingress %s/%s: %v",
					retryIR.GetSource().IngressNN.Namespace, retryIR.GetSource().IngressNN.Name, err),
					&ctx.HTTPRoute)
				continue
			}
			backendTrafficPolicy.Spec.Retry.RetryOn = &egapiv1a1.RetryOn{
				Triggers:        triggers,
				HTTPStatusCodes: statusCodes,
			}
			backendTrafficPolicy.Spec.Retry.NumRetries = retryIR.Count

			retryIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted Retry annotations of ingress %s/%s",
				retryIR.GetSource().IngressNN.Namespace, retryIR.GetSource().IngressNN.Name),
				&ctx.HTTPRoute)
		}
	}
}

func (e *Emitter) convertRetryTriggersToEnvoy(triggers []string) ([]egapiv1a1.TriggerEnum, []egapiv1a1.HTTPStatus, error) {
	triggerSet := sets.New[egapiv1a1.TriggerEnum]()
	invalidTriggerSet := sets.New[string]()
	statusSet := sets.New[egapiv1a1.HTTPStatus]()

	for _, trigger := range triggers {
		switch trigger {
		case "error":
			triggerSet.Insert(egapiv1a1.ConnectFailure)
			triggerSet.Insert(egapiv1a1.ResetBeforeRequest)
			triggerSet.Insert(egapiv1a1.RefusedStream)
		case "timeout":
			triggerSet.Insert(egapiv1a1.ConnectFailure)
			triggerSet.Insert(egapiv1a1.Reset)
		case "http_500":
			triggerSet.Insert(egapiv1a1.RetriableStatusCodes)
			statusSet.Insert(500)
		case "http_502":
			triggerSet.Insert(egapiv1a1.RetriableStatusCodes)
			statusSet.Insert(502)
		case "http_503":
			triggerSet.Insert(egapiv1a1.RetriableStatusCodes)
			statusSet.Insert(503)
		case "http_504":
			triggerSet.Insert(egapiv1a1.RetriableStatusCodes)
			statusSet.Insert(504)
		case "http_429":
			triggerSet.Insert(egapiv1a1.RetriableStatusCodes)
			statusSet.Insert(429)
		case "http_403", "http_404":
			// The cases of http_403 and http_404 are never considered unsuccessful attempts.
			// https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_next_upstream
			continue
		default:
			invalidTriggerSet.Insert(trigger)
		}
	}

	if len(invalidTriggerSet) > 0 {
		return nil, nil, fmt.Errorf("unsupported retry trigger: %s", invalidTriggerSet.UnsortedList())
	}
	return triggerSet.UnsortedList(), statusSet.UnsortedList(), nil
}
