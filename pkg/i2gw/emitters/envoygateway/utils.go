package envoygateway_emitter

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"k8s.io/utils/ptr"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
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
		port, err := strconv.ParseInt(parsedURL.Port(), 10, 32)
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

// parseServiceURL parses a Kubernetes service URL and returns a BackendObjectReference.
// Supports multiple hostname formats:
//   - <service-name>.<namespace>.svc.cluster.local
//   - <service-name>.<namespace>.svc
//   - <service-name>.<namespace>
//   - <service-name> (uses defaultNamespace)
//
// URL format: http(s)://<hostname>:<port>/<path>
func parseK8sServiceURL(urlStr string, defaultNamespace string) (*gwapiv1.BackendObjectReference, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Extract hostname and port
	hostname := parsedURL.Hostname()
	parts := strings.Split(hostname, ".")

	var serviceName, namespace string

	switch {
	case strings.HasSuffix(hostname, ".svc.cluster.local"):
		// Format: <service-name>.<namespace>.svc.cluster.local
		if len(parts) < 5 {
			return nil, fmt.Errorf("invalid service URL format: %s", hostname)
		}
		serviceName = parts[0]
		namespace = parts[1]

	case strings.HasSuffix(hostname, ".svc"):
		// Format: <service-name>.<namespace>.svc
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid service URL format: %s", hostname)
		}
		serviceName = parts[0]
		namespace = parts[1]

	case len(parts) == 2:
		// Format: <service-name>.<namespace>
		serviceName = parts[0]
		namespace = parts[1]

	case len(parts) == 1:
		// Format: <service-name> (use default namespace)
		serviceName = parts[0]
		namespace = defaultNamespace

	default:
		return nil, fmt.Errorf("unsupported service URL format: %s", hostname)
	}

	// Extract port
	var port gwapiv1.PortNumber
	if parsedURL.Port() != "" {
		portNum, err := strconv.ParseInt(parsedURL.Port(), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid port number: %w", err)
		}
		port = gwapiv1.PortNumber(portNum)
	} else {
		// Default ports
		if parsedURL.Scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	}

	return &gwapiv1.BackendObjectReference{
		Group:     ptr.To(gwapiv1.Group("")),
		Kind:      ptr.To(gwapiv1.Kind("Service")),
		Name:      gwapiv1.ObjectName(serviceName),
		Namespace: ptr.To(gwapiv1.Namespace(namespace)),
		Port:      ptr.To(port),
	}, nil
}
