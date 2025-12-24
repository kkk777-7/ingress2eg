package envoygateway_emitter

import (
	"fmt"

	egapiv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
)

func builderRewriteHTTPRouteFilter(
	ctx emitterir.HTTPRouteContext,
	ruleIdx int,
	regexIR *emitterir.RegexFeatureIR,
	rewriteIR *emitterir.RewriteFeatureIR,
) *egapiv1a1.HTTPRouteFilter {
	filter := &egapiv1a1.HTTPRouteFilter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", ctx.Name, ruleIdx),
			Namespace: ctx.Namespace,
		},
		Spec: egapiv1a1.HTTPRouteFilterSpec{
			URLRewrite: &egapiv1a1.HTTPURLRewriteFilter{
				Path: &egapiv1a1.HTTPPathModifier{
					Type: egapiv1a1.RegexHTTPPathModifier,
					ReplaceRegexMatch: &egapiv1a1.ReplaceRegexMatch{
						Pattern:      fmt.Sprintf("^%s$", regexIR.PathPattern),
						Substitution: convertNginxSubstitutionToEnvoy(rewriteIR.Target),
					},
				},
			},
		},
	}
	filter.SetGroupVersionKind(HTTPRouteFilterGVK)
	return filter
}
