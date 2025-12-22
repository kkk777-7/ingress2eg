# Ingress to Envoy Gateway

ingress2eg helps translate Ingress to Gateway API and Envoy Gateway CRD resources.

## ⚠️ Important Notice

**This is an unofficial POC tool forked from [ingress2gateway](https://github.com/kubernetes-sigs/ingress2gateway).**

- This tool is a **temporary solution** until official support is available
- aim to backport features to the official ingress2gateway project
- **Once official support is complete, strongly recommend using ingress2gateway instead**

## 🎯 Scope

This project primarily focuses on converting NGINX Ingress resources to Envoy Gateway, in response to the [retirement of ingress-nginx](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/). The main goal is to provide a migration path for users transitioning from NGINX Ingress to Envoy Gateway.

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