package envoygateway_emitter

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/emitters/utils"
)

const emitterName = "envoy-gateway"

func init() {
	i2gw.EmitterConstructorByName[emitterName] = NewEmitter
}

type Emitter struct{}

func NewEmitter(_ *i2gw.EmitterConf) i2gw.Emitter {
	return &Emitter{}
}

func (c *Emitter) Emit(ir emitterir.EmitterIR) (i2gw.GatewayResources, field.ErrorList) {
	for key, ctx := range ir.HTTPRoutes {
		ir.HTTPRoutes[key] = ctx
	}

	// NOTE:
	// If common emitter will implement, should remove `utils.ToGatewayResources`.
	// Envoy Gateway Emitter should only handle custom resources generation.
	gatewayResources, errs := utils.ToGatewayResources(ir)
	if len(errs) != 0 {
		return i2gw.GatewayResources{}, errs
	}
	c.ToEnvoyGatewayResources(ir, &gatewayResources)

	return gatewayResources, nil
}

func (c *Emitter) ToEnvoyGatewayResources(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	c.EmitRegex(ir, gwResources)
}
