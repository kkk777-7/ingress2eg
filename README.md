# Ingress to Envoy Gateway

ingress2eg helps translate Ingress to Gateway API and Envoy Gateway CRD resources.

## ⚠️ Important Notice

**This is an unofficial POC tool forked from [ingress2gateway](https://github.com/kubernetes-sigs/ingress2gateway).**

- This tool is a **temporary solution** until official support is available
- aim to backport features to the official ingress2gateway project
- **Once official support is complete, strongly recommend using ingress2gateway instead**

## 🎯 Scope

This project primarily focuses on converting NGINX Ingress resources to Envoy Gateway, in response to the [retirement of ingress-nginx](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/). The main goal is to provide a migration path for users transitioning from NGINX Ingress to Envoy Gateway.

## 🚀 Supported Features

This tool supports converting the following NGINX Ingress annotations to Envoy Gateway resources:

| Feature Category | NGINX Ingress Annotations | Envoy Gateway Resources | Description |
|-----------------|---------------------------|------------------------|-------------|
| **Session Affinity** | [`affinity`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#session-affinity), [`session-cookie-name`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#cookie-affinity), [`session-cookie-max-age`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#cookie-affinity), [`session-cookie-expires`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#cookie-affinity), [`session-cookie-samesite`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#cookie-affinity) | `BackendTrafficPolicy` ([LoadBalancer.ConsistentHash](https://gateway.envoyproxy.io/docs/api/extension_types/#consistenthash)) | Cookie-based session affinity for sticky sessions |
| **Authentication - Basic** | [`auth-type`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#authentication), [`auth-secret`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#authentication)<br>[\*1](#note-1) | `SecurityPolicy` ([BasicAuth](https://gateway.envoyproxy.io/docs/api/extension_types/#basicauth)) | HTTP Basic Authentication |
| **Authentication - mTLS** | [`auth-tls-secret`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#client-certificate-authentication) | `ClientTrafficPolicy` ([ClientTLSSettings](https://gateway.envoyproxy.io/docs/api/extension_types/#clienttlssettings)) | Mutual TLS authentication at Gateway listener level |
| **Authentication - External** | [`auth-url`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#external-authentication) [\*2](#note-2), [`auth-response-headers`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#external-authentication) | `SecurityPolicy` ([ExtAuth](https://gateway.envoyproxy.io/docs/api/extension_types/#extauth)) | External authentication service integration |
| **Backend TLS** | [`proxy-ssl-secret`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#backend-certificate-authentication), [`proxy-ssl-verify`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#backend-certificate-authentication), [`proxy-ssl-name`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#backend-certificate-authentication), [`proxy-ssl-server-name`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#backend-certificate-authentication) | `Backend` ([BackendTLSSettings](https://gateway.envoyproxy.io/docs/api/extension_types/#backendtlssettings)) | TLS configuration for upstream connections |
| **Backend Protocol** | [`backend-protocol`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#backend-protocol) | `HTTPRoute` / `GRPCRoute` | Protocol detection and route type conversion (HTTP/GRPC) |
| **Buffer Limits** | [`proxy-body-size`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#custom-max-body-size) | `BackendTrafficPolicy` ([BackendConnection.BufferLimit](https://gateway.envoyproxy.io/docs/api/extension_types/#backendconnection)) | Client request buffer size limits |
| **Canary Deployment** | [`canary`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#canary), [`canary-by-header`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#canary), [`canary-by-header-value`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#canary), [`canary-weight`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#canary), [`canary-weight-total`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#canary) | `HTTPRoute` ([HTTPRouteRule](https://gateway-api.sigs.k8s.io/reference/1.4/spec/#httprouterule) with weights/matches) | Header-based and weight-based traffic splitting |
| **CORS** | [`enable-cors`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#enable-cors), [`cors-allow-origin`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#enable-cors), [`cors-allow-methods`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#enable-cors), [`cors-allow-headers`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#enable-cors), [`cors-expose-headers`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#enable-cors), [`cors-max-age`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#enable-cors), [`cors-allow-credentials`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#enable-cors) | `SecurityPolicy` ([CORS](https://gateway.envoyproxy.io/docs/api/extension_types/#cors)) | Cross-Origin Resource Sharing policy |
| **Header Modification** | [`x-forwarded-prefix`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#x-forwarded-prefix-header), [`upstream-vhost`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#custom-nginx-upstream-vhost) | `HTTPRoute` ([RequestHeaderModifier](https://gateway-api.sigs.k8s.io/reference/1.4/spec/#httproutefilter) filter) | Request header manipulation |
| **IP Range Control** | [`whitelist-source-range`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#whitelist-source-range), [`denylist-source-range`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#denylist-source-range) | `SecurityPolicy` ([Authorization](https://gateway.envoyproxy.io/docs/api/extension_types/#authorization)) | IP-based access control with allowlist/denylist |
| **Rate Limiting** | [`limit-rps`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#rate-limiting), [`limit-rpm`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#rate-limiting) | `BackendTrafficPolicy` ([RateLimit.Local](https://gateway.envoyproxy.io/docs/api/extension_types/#localratelimit)) | Request rate limiting per client IP |
| **Redirect** | [`ssl-redirect`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#server-side-https-enforcement-through-redirect), [`force-ssl-redirect`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#server-side-https-enforcement-through-redirect), [`permanent-redirect`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#permanent-redirect), [`temporal-redirect`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#temporal-redirect), [`app-root`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#rewrite) | `HTTPRoute` ([RequestRedirect](https://gateway-api.sigs.k8s.io/reference/1.4/spec/#httproutefilter) filter) | HTTP to HTTPS and URL redirects |
| **Regex Path Matching** | [`use-regex`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#use-regex) | `HTTPRoute` ([PathMatchRegularExpression](https://gateway-api.sigs.k8s.io/reference/1.4/spec/#httppathmatch)) | Regular expression path matching |
| **Retry Policy** | [`proxy-next-upstream`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#custom-timeouts), [`proxy-next-upstream-tries`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#custom-timeouts) | `BackendTrafficPolicy` ([Retry](https://gateway.envoyproxy.io/docs/api/extension_types/#retry)) | Automatic request retry on failures |
| **URL Rewrite** | [`rewrite-target`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#rewrite) | `HTTPRoute` ([URLRewrite](https://gateway-api.sigs.k8s.io/reference/1.4/spec/#httproutefilter) filter) or [`HTTPRouteFilter`](https://gateway.envoyproxy.io/docs/api/extension_types/#httpurlrewritefilter) (ExtensionRef) | Path rewriting with or without regex |
| **SSL Passthrough** | [`ssl-passthrough`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#ssl-passthrough) | `TLSRoute` | TLS passthrough mode without termination |
| **Timeout** | [`proxy-connect-timeout`](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/annotations/#custom-timeouts) | `BackendTrafficPolicy` ([Timeout.TCP](https://gateway.envoyproxy.io/docs/api/extension_types/#tcptimeout)) | Connection timeout configuration |

### Notes

<a id="note-1">**\*1 Basic Authentication Requirements:**</a>
- Only `basic` auth-type is supported. Other authentication types (e.g., digest) are not converted.
- For Envoy Gateway, the Secret must contain a key named `.htpasswd` in htpasswd format.
- Example Secret format:
  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: basic-auth-secret
  type: Opaque
  data:
    .htpasswd: <base64-encoded-htpasswd-content>
  ```
<a id="note-2">**\*2 External Authentication Requirements:**</a>
- Only kubernetes service endpoint is supported in `auth-url`. (e.g., `http://auth-service.namespace.svc.cluster.local/auth`)

## 🚀 Quick Start

### Installation

#### Option 1: Install directly with go install

```bash
go install github.com/kkk777-7/ingress2eg@latest
```

#### Option 2: Build from source

```bash
git clone https://github.com/kkk777-7/ingress2eg.git
cd ingress2eg
make build
```

### Basic Usage

#### Convert from Kubernetes Cluster

Convert all Ingress resources in the current namespace:

```bash
ingress2eg print
```

Convert from a specific namespace:

```bash
ingress2eg print --namespace myapp
```

Convert from all namespaces:

```bash
ingress2eg print --all-namespaces
```

#### Convert from File

Convert Ingress resources from a YAML or JSON file:

```bash
ingress2eg print --input-file ingress.yaml
```

Save the output to a file:

```bash
ingress2eg print --input-file ingress.yaml > gateway-resources.yaml
```

#### Advanced Options

Specify a custom Ingress class (default is `nginx`):

```bash
ingress2eg print --ingress-nginx-ingress-class=custom-nginx
```

Change output format:

```bash
# Output as JSON
ingress2eg print --output json

# Output as YAML (default)
ingress2eg print --output yaml
```

### Example

Given an Ingress resource with NGINX annotations:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-app
  namespace: default
  annotations:
    nginx.ingress.kubernetes.io/affinity: "cookie"
    nginx.ingress.kubernetes.io/session-cookie-name: "route"
    nginx.ingress.kubernetes.io/cors-allow-origin: "*"
    nginx.ingress.kubernetes.io/enable-cors: "true"
spec:
  ingressClassName: nginx
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: example-service
            port:
              number: 80
```

Run the conversion:

```bash
ingress2eg print --input-file ingress.yaml
```

This will generate:
- `Gateway` resource
- `HTTPRoute` resource
- `BackendTrafficPolicy` with session affinity configuration
- `SecurityPolicy` with CORS configuration

The tool will display informational messages showing which annotations were parsed and converted:

```
parsed Affinity (affinity, session-cookie-name) of ingress default/example-app
parsed CORS (enable-cors, cors-allow-origin) of ingress default/example-app
converted Affinity annotations of ingress default/example-app for rule rule-0
converted CORS annotations of ingress default/example-app for rule rule-0
```

## 📚 References

### Gateway API
- [Gateway API Specification](https://gateway-api.sigs.k8s.io/)
- [Gateway API GEPs](https://gateway-api.sigs.k8s.io/geps/overview/)

### NGINX Ingress
- [NGINX Ingress Controller Documentation](https://kubernetes.github.io/ingress-nginx/)
- [NGINX Ingress Controller GitHub](https://github.com/kubernetes/ingress-nginx)
- [Retirement Announcement](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/)

### Envoy Gateway
- [Envoy Gateway Documentation](https://gateway.envoyproxy.io/)
- [Envoy Gateway GitHub](https://github.com/envoyproxy/gateway)