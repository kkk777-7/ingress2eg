package envoygateway_emitter

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kkk777-7/ingress2eg/pkg/i2gw"
	emitterir "github.com/kkk777-7/ingress2eg/pkg/i2gw/emitter_intermediate"
	"github.com/kkk777-7/ingress2eg/pkg/i2gw/notifications"
)

// redirectURLComponents represents the parsed components of a redirect URL
type redirectURLComponents struct {
	scheme   *string
	hostname *gwapiv1.PreciseHostname
	port     *gwapiv1.PortNumber
	path     *gwapiv1.HTTPPathModifier
}

// parseRedirectURL parses a redirect URL string into Gateway API components.
// Expects a full URL with scheme (e.g., "https://example.com/path").
func parseRedirectURL(urlStr string) (*redirectURLComponents, error) {
	components := &redirectURLComponents{}

	// Validate that URL contains a scheme
	if !strings.Contains(urlStr, "://") {
		return nil, fmt.Errorf("URL must include scheme (e.g., https://), got: %s", urlStr)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Validate scheme
	if parsedURL.Scheme == "" {
		return nil, fmt.Errorf("URL must include scheme (e.g., https://)")
	}

	// Validate hostname
	if parsedURL.Hostname() == "" {
		return nil, fmt.Errorf("URL must include hostname")
	}

	// Set scheme
	components.scheme = ptr.To(parsedURL.Scheme)

	// Set hostname
	hostname := gwapiv1.PreciseHostname(parsedURL.Hostname())
	components.hostname = &hostname

	// Set port
	if parsedURL.Port() != "" {
		port, err := strconv.Atoi(parsedURL.Port())
		if err != nil {
			return nil, fmt.Errorf("failed to parse port: %w", err)
		}
		portNum := gwapiv1.PortNumber(port)
		components.port = &portNum
	}

	// Set path (only if it's not just "/")
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		components.path = &gwapiv1.HTTPPathModifier{
			Type:            gwapiv1.FullPathHTTPPathModifier,
			ReplaceFullPath: ptr.To(parsedURL.Path),
		}
	}
	return components, nil
}

func (c *Emitter) EmitRedirect(ir emitterir.EmitterIR, gwResources *i2gw.GatewayResources) {
	for _, ctx := range ir.HTTPRoutes {
		for idx, ir := range ctx.ExtensionFeatures[emitterir.RedirectFeatureKey] {
			if ir.IsParsed() {
				continue
			}
			redirectIR := ir.(*emitterir.RedirectFeatureIR)

			nn := types.NamespacedName{
				Namespace: ctx.Namespace,
				Name:      ctx.Name,
			}
			route := gwResources.HTTPRoutes[nn]

			if len(route.Spec.ParentRefs) == 0 {
				notify(notifications.ErrorNotification, fmt.Sprintf("Cannot convert Redirect annotation for Ingress %s/%s: no parent refs found",
					redirectIR.GetSource().IngressNN.Namespace, redirectIR.GetSource().IngressNN.Name),
					&ctx.HTTPRoute)
				continue
			}

			gwNN := types.NamespacedName{
				Namespace: string(ptr.Deref(route.Spec.ParentRefs[0].Namespace, gwapiv1.Namespace(ctx.Namespace))),
				Name:      string(route.Spec.ParentRefs[0].Name),
			}
			gw := gwResources.Gateways[gwNN]

			c.emitSslRedirect(&route, &gw, redirectIR, idx)
			c.emitRedirect(&route, redirectIR, idx)

			gwResources.HTTPRoutes[nn] = route

			redirectIR.SetParsed()
			notify(notifications.InfoNotification, fmt.Sprintf("converted Redirect annotations of ingress %s/%s",
				redirectIR.GetSource().IngressNN.Namespace, redirectIR.GetSource().IngressNN.Name),
				&ctx.HTTPRoute)
		}
	}
}

func (c *Emitter) emitSslRedirect(route *gwapiv1.HTTPRoute, gw *gwapiv1.Gateway, redirectIR *emitterir.RedirectFeatureIR, ruleIdx int) {
	if redirectIR.Ssl == nil {
		return
	}

	if redirectIR.Ssl.Force {
		filter := gwapiv1.HTTPRequestRedirectFilter{
			Scheme:     ptr.To("https"),
			StatusCode: ptr.To(301),
		}
		route.Spec.Rules[ruleIdx].Filters = append(route.Spec.Rules[ruleIdx].Filters,
			gwapiv1.HTTPRouteFilter{
				Type:            gwapiv1.HTTPRouteFilterRequestRedirect,
				RequestRedirect: &filter,
			},
		)
	} else {
		var foundHTTPSListener bool
		for _, listener := range gw.Spec.Listeners {
			if listener.Protocol == gwapiv1.HTTPSProtocolType {
				foundHTTPSListener = true
				break
			}
		}

		if foundHTTPSListener {
			filter := gwapiv1.HTTPRequestRedirectFilter{
				Scheme:     ptr.To("https"),
				StatusCode: ptr.To(301),
			}
			route.Spec.Rules[ruleIdx].Filters = append(route.Spec.Rules[ruleIdx].Filters,
				gwapiv1.HTTPRouteFilter{
					Type:            gwapiv1.HTTPRouteFilterRequestRedirect,
					RequestRedirect: &filter,
				},
			)
		}
	}
}

func (c *Emitter) emitRedirect(route *gwapiv1.HTTPRoute, redirectIR *emitterir.RedirectFeatureIR, ruleIdx int) {
	if redirectIR.Url == "" {
		return
	}

	components, err := parseRedirectURL(redirectIR.Url)
	if err != nil {
		notify(notifications.ErrorNotification, fmt.Sprintf("Cannot convert Redirect annotation for Ingress %s/%s: %v",
			redirectIR.GetSource().IngressNN.Namespace, redirectIR.GetSource().IngressNN.Name, err),
			route)
		return
	}

	// Build the redirect filter
	filter := gwapiv1.HTTPRequestRedirectFilter{
		Scheme:     components.scheme,
		Hostname:   components.hostname,
		Port:       components.port,
		Path:       components.path,
		StatusCode: ptr.To(redirectIR.StatusCode),
	}

	route.Spec.Rules[ruleIdx].Filters = append(route.Spec.Rules[ruleIdx].Filters,
		gwapiv1.HTTPRouteFilter{
			Type:            gwapiv1.HTTPRouteFilterRequestRedirect,
			RequestRedirect: &filter,
		},
	)
}
