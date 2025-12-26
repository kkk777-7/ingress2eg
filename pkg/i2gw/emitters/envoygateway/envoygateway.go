package envoygateway_emitter

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/emitters/utils"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

const emitterName = "envoy-gateway"

func init() {
	i2gw.EmitterConstructorByName[emitterName] = NewEmitter
}

type Emitter struct {
	builderMap *BuilderMap
}

func NewEmitter(_ *i2gw.EmitterConf) i2gw.Emitter {
	return &Emitter{
		builderMap: NewBuilderMap(),
	}
}

func (e *Emitter) Emit(ir emitterir.EmitterIR) (i2gw.GatewayResources, field.ErrorList) {
	// NOTE:
	// If common emitter will implement, should remove `utils.ToGatewayResources`.
	// Envoy Gateway Emitter should only handle custom resources generation.
	gatewayResources, errs := utils.ToGatewayResources(ir)
	if len(errs) != 0 {
		return i2gw.GatewayResources{}, errs
	}
	e.ToEnvoyGatewayResources(ir, &gatewayResources)

	return gatewayResources, nil
}

func (e *Emitter) ToEnvoyGatewayResources(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	e.EmitRegex(ir, gwResources)
	e.EmitRewrite(ir, gwResources)
	e.EmitRedirect(ir, gwResources)
	e.EmitBasicAuth(ir, gwResources)
	e.EmitMTLS(ir, gwResources)
	e.EmitExternalAuth(ir, gwResources)
	e.EmitIpRange(ir, gwResources)
	e.EmitRateLimit(ir, gwResources)
	e.EmitBuffer(ir, gwResources)
	e.EmitCors(ir, gwResources)
	e.EmitBackendTLS(ir, gwResources)

	for _, backend := range e.builderMap.Backends {
		obj, err := i2gw.CastToUnstructured(backend)
		if err != nil {
			notify(notifications.ErrorNotification, "Failed to cast Backend to unstructured", backend)
			continue
		}
		gwResources.GatewayExtensions = append(gwResources.GatewayExtensions, *obj)
	}
	for _, securityPolicy := range e.builderMap.SecurityPolicies {
		obj, err := i2gw.CastToUnstructured(securityPolicy)
		if err != nil {
			notify(notifications.ErrorNotification, "Failed to cast SecurityPolicy to unstructured", securityPolicy)
			continue
		}
		gwResources.GatewayExtensions = append(gwResources.GatewayExtensions, *obj)
	}
	for _, clientTrafficPolicy := range e.builderMap.ClientTrafficPolicies {
		obj, err := i2gw.CastToUnstructured(clientTrafficPolicy)
		if err != nil {
			notify(notifications.ErrorNotification, "Failed to cast ClientTrafficPolicy to unstructured", clientTrafficPolicy)
			continue
		}
		gwResources.GatewayExtensions = append(gwResources.GatewayExtensions, *obj)
	}
	for _, backendTrafficPolicy := range e.builderMap.BackendTrafficPolicies {
		obj, err := i2gw.CastToUnstructured(backendTrafficPolicy)
		if err != nil {
			notify(notifications.ErrorNotification, "Failed to cast BackendTrafficPolicy to unstructured", backendTrafficPolicy)
			continue
		}
		gwResources.GatewayExtensions = append(gwResources.GatewayExtensions, *obj)
	}
}
