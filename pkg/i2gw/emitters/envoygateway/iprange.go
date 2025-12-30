package envoygateway_emitter

import (
	"fmt"
	"slices"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"github.com/samber/lo"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

func (e *Emitter) EmitIpRange(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		ctx.MergeExtensionFeature(emitterir.IpRangeFeatureKey)

		for idx, ir := range ctx.ExtensionFeatures[emitterir.IpRangeFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			ipRangeIR := ir.(*emitterir.IpRangeFeatureIR)

			var sectionName *gwapiv1.SectionName
			if idx != emitterir.RouteRuleAllIndex && idx < len(ctx.Spec.Rules) {
				sectionName = ctx.Spec.Rules[idx].Name
			}
			securityPolicy := e.getOrBuildSecurityPolicy(ctx, sectionName, idx)

			if len(ipRangeIR.AllowList) > 0 {
				slices.Sort(ipRangeIR.AllowList)
				securityPolicy.Spec.Authorization = &egapiv1a1.Authorization{
					DefaultAction: ptr.To(egapiv1a1.AuthorizationActionDeny),
					Rules: []egapiv1a1.AuthorizationRule{
						{
							Action: egapiv1a1.AuthorizationActionAllow,
							Principal: egapiv1a1.Principal{
								ClientCIDRs: lo.Map(ipRangeIR.AllowList, func(a string, _ int) egapiv1a1.CIDR {
									return egapiv1a1.CIDR(a)
								}),
							},
						},
					},
				}
			}
			if len(ipRangeIR.DenyList) > 0 {
				slices.Sort(ipRangeIR.DenyList)
				securityPolicy.Spec.Authorization = &egapiv1a1.Authorization{
					DefaultAction: ptr.To(egapiv1a1.AuthorizationActionAllow),
					Rules: []egapiv1a1.AuthorizationRule{
						{
							Action: egapiv1a1.AuthorizationActionDeny,
							Principal: egapiv1a1.Principal{
								ClientCIDRs: lo.Map(ipRangeIR.DenyList, func(a string, _ int) egapiv1a1.CIDR {
									return egapiv1a1.CIDR(a)
								}),
							},
						},
					},
				}
			}

			ipRangeIR.SetParsed()
			ruleInfo := ""
			if sectionName != nil {
				ruleInfo = fmt.Sprintf(" for rule %s", *sectionName)
			}
			notify(notifications.InfoNotification, fmt.Sprintf("converted IPRange annotations of ingress %s/%s%s",
				ipRangeIR.GetSource().IngressNN.Namespace, ipRangeIR.GetSource().IngressNN.Name, ruleInfo),
				&ctx.HTTPRoute)
		}
	}
}
