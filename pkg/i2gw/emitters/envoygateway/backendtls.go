package envoygateway_emitter

import (
	"fmt"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

func (e *Emitter) EmitBackendTLS(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	// Process HTTPRoutes
	for _, ctx := range ir.HTTPRoutes {
		ctx.MergeExtensionFeature(emitterir.BackendTLSFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.BackendTLSFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			backendTLSIR := ir.(*emitterir.BackendTLSFeatureIR)

			nn := types.NamespacedName{
				Namespace: ctx.Namespace,
				Name:      ctx.Name,
			}
			route := gwResources.HTTPRoutes[nn]

			e.processBackendTLSForHTTPRoute(&route, backendTLSIR, idx)

			gwResources.HTTPRoutes[nn] = route

			backendTLSIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted BackendTLS annotations of ingress %s/%s",
				backendTLSIR.GetSource().IngressNN.Namespace, backendTLSIR.GetSource().IngressNN.Name),
				&ctx.HTTPRoute)
		}
	}

	// Process GRPCRoutes
	for _, ctx := range ir.GRPCRoutes {
		ctx.MergeExtensionFeature(emitterir.BackendTLSFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.BackendTLSFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			backendTLSIR := ir.(*emitterir.BackendTLSFeatureIR)

			nn := types.NamespacedName{
				Namespace: ctx.Namespace,
				Name:      ctx.Name,
			}
			route := gwResources.GRPCRoutes[nn]

			e.processBackendTLSForGRPCRoute(&route, backendTLSIR, idx)

			gwResources.GRPCRoutes[nn] = route

			backendTLSIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted BackendTLS annotations of ingress %s/%s",
				backendTLSIR.GetSource().IngressNN.Namespace, backendTLSIR.GetSource().IngressNN.Name),
				&ctx.GRPCRoute)
		}
	}
}

func (e *Emitter) processBackendTLSForHTTPRoute(route *gwapiv1.HTTPRoute, backendTLSIR *emitterir.BackendTLSFeatureIR, ruleIdx int) {
	var targetIdx []int
	if ruleIdx == emitterir.RouteRuleAllIndex {
		for i := range route.Spec.Rules {
			targetIdx = append(targetIdx, i)
		}
	} else {
		targetIdx = []int{ruleIdx}
	}

	for _, idx := range targetIdx {
		for i, backendRef := range route.Spec.Rules[idx].BackendRefs {
			nn := types.NamespacedName{
				Namespace: route.Namespace,
				Name:      route.Name,
			}
			backend := e.getOrBuildBackend(nn, ruleIdx, backendRef.BackendObjectReference)

			e.configureBackendTLS(backend, backendTLSIR)

			// Update the HTTPRoute's backendRef to point to the Backend instead of the Service
			route.Spec.Rules[idx].BackendRefs[i].BackendObjectReference = gwapiv1.BackendObjectReference{
				Group: ptr.To(gwapiv1.Group(BackendGVK.Group)),
				Kind:  ptr.To(gwapiv1.Kind(BackendGVK.Kind)),
				Name:  gwapiv1.ObjectName(backend.Name),
			}
		}
	}
}

func (e *Emitter) processBackendTLSForGRPCRoute(route *gwapiv1.GRPCRoute, backendTLSIR *emitterir.BackendTLSFeatureIR, ruleIdx int) {
	var targetIdx []int
	if ruleIdx == emitterir.RouteRuleAllIndex {
		for i := range route.Spec.Rules {
			targetIdx = append(targetIdx, i)
		}
	} else {
		targetIdx = []int{ruleIdx}
	}

	for _, idx := range targetIdx {
		for i, backendRef := range route.Spec.Rules[idx].BackendRefs {
			nn := types.NamespacedName{
				Namespace: route.Namespace,
				Name:      route.Name,
			}
			backend := e.getOrBuildBackend(nn, ruleIdx, backendRef.BackendObjectReference)

			e.configureBackendTLS(backend, backendTLSIR)

			// Update the GRPCRoute's backendRef to point to the Backend instead of the Service
			route.Spec.Rules[idx].BackendRefs[i].BackendObjectReference = gwapiv1.BackendObjectReference{
				Group: ptr.To(gwapiv1.Group(BackendGVK.Group)),
				Kind:  ptr.To(gwapiv1.Kind(BackendGVK.Kind)),
				Name:  gwapiv1.ObjectName(backend.Name),
			}
		}
	}
}

func (e *Emitter) configureBackendTLS(backend *egapiv1a1.Backend, backendTLSIR *emitterir.BackendTLSFeatureIR) {
	if backend.Spec.TLS == nil {
		backend.Spec.TLS = &egapiv1a1.BackendTLSSettings{}
	}
	backend.Spec.TLS.CACertificateRefs = []gwapiv1.LocalObjectReference{backendTLSIR.CertificateRef}
	backend.Spec.TLS.BackendTLSConfig = &egapiv1a1.BackendTLSConfig{
		ClientCertificateRef: &gwapiv1.SecretObjectReference{
			Group: ptr.To(gwapiv1.Group(SecretGVK.Group)),
			Kind:  ptr.To(gwapiv1.Kind(SecretGVK.Kind)),
			Name:  backendTLSIR.CertificateRef.Name,
		},
	}
	if backendTLSIR.Sni != "" {
		backend.Spec.TLS.SNI = ptr.To(gwapiv1.PreciseHostname(backendTLSIR.Sni))
	}
	if backendTLSIR.InsecureSkipVerify != nil {
		backend.Spec.TLS.InsecureSkipVerify = backendTLSIR.InsecureSkipVerify
	}
}
