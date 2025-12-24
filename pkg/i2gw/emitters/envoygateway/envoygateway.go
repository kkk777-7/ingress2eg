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
	builderMap *builderMap
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

	for _, securityPolicy := range e.builderMap.SecurityPolices {
		obj, err := i2gw.CastToUnstructured(securityPolicy)
		if err != nil {
			notify(notifications.ErrorNotification, "Failed to cast SecurityPolicy to unstructured", securityPolicy)
			continue
		}
		gwResources.GatewayExtensions = append(gwResources.GatewayExtensions, *obj)
	}
}
