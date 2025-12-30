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
| **Session Affinity** | `affinity`, `session-cookie-name`, `session-cookie-max-age`, `session-cookie-expires`, `session-cookie-samesite` | `BackendTrafficPolicy` (LoadBalancer.ConsistentHash) | Cookie-based session affinity for sticky sessions |
| **Authentication - Basic** | `auth-type`, `auth-secret` | `SecurityPolicy` (BasicAuth) | HTTP Basic Authentication |
| **Authentication - mTLS** | `auth-tls-secret` | `ClientTrafficPolicy` (TLS.ClientValidation) | Mutual TLS authentication at Gateway listener level |
| **Authentication - External** | `auth-url`, `auth-response-headers` | `SecurityPolicy` (ExtAuth) | External authentication service integration |
| **Backend TLS** | `proxy-ssl-secret`, `proxy-ssl-verify`, `proxy-ssl-name`, `proxy-ssl-server-name` | `Backend` (TLS) | TLS configuration for upstream connections |
| **Backend Protocol** | `backend-protocol` | `HTTPRoute` / `GRPCRoute` | Protocol detection and route type conversion (HTTP/GRPC) |
| **Buffer Limits** | `proxy-body-size` | `ClientTrafficPolicy` (Connection.BufferLimit) | Client request buffer size limits |
| **Canary Deployment** | `canary`, `canary-by-header`, `canary-by-header-value`, `canary-weight`, `canary-weight-total` | `HTTPRoute` (HTTPRouteRule with weights/matches) | Header-based and weight-based traffic splitting |
| **CORS** | `enable-cors`, `cors-allow-origin`, `cors-allow-methods`, `cors-allow-headers`, `cors-expose-headers`, `cors-max-age`, `cors-allow-credentials` | `SecurityPolicy` (CORS) | Cross-Origin Resource Sharing policy |
| **Header Modification** | `x-forwarded-prefix`, `upstream-vhost` | `HTTPRoute` (RequestHeaderModifier filter) | Request header manipulation |
| **IP Range Control** | `whitelist-source-range`, `denylist-source-range` | `SecurityPolicy` (Authorization) | IP-based access control with allowlist/denylist |
| **Rate Limiting** | `limit-rps`, `limit-rpm` | `BackendTrafficPolicy` (RateLimit.Local) | Request rate limiting per client IP |
| **Redirect** | `ssl-redirect`, `force-ssl-redirect`, `permanent-redirect`, `temporal-redirect` | `HTTPRoute` (RequestRedirect filter) | HTTP to HTTPS and URL redirects |
| **Regex Path Matching** | `use-regex` | `HTTPRoute` (PathMatchRegularExpression) | Regular expression path matching |
| **Retry Policy** | `proxy-next-upstream`, `proxy-next-upstream-tries` | `BackendTrafficPolicy` (Retry) | Automatic request retry on failures |
| **URL Rewrite** | `rewrite-target`, `app-root` | `HTTPRoute` (URLRewrite filter) or `HTTPRouteFilter` (ExtensionRef) | Path rewriting with or without regex |
| **SSL Passthrough** | `ssl-passthrough` | `TLSRoute` | TLS passthrough mode without termination |
| **Timeout** | `proxy-connect-timeout` | `BackendTrafficPolicy` (Timeout.TCP) | Connection timeout configuration |

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