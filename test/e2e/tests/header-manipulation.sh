#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/header-manipulation.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/header-manipulation.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/header-manipulation.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/header-manipulation.yaml

echo -e ""
echo -e "Waiting for IPs..."
echo -e ""

# wait for INGRESS_IP with timeout
TIMEOUT=300  # 5 minutes
ELAPSED=0

INGRESS_IP=""
while [ -z "$INGRESS_IP" ] && [ $ELAPSED -lt $TIMEOUT ]; do
  echo -e "Waiting for INGRESS_IP... (${ELAPSED}s/${TIMEOUT}s)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  INGRESS_IP=$(kubectl get ingress -n ingress2eg upstream-vhost -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
done

if [ -z "$INGRESS_IP" ]; then
  echo -e "❌ Error: Timeout waiting for INGRESS_IP"
  exit 1
fi

# wait for EG_IP with timeout
ELAPSED=0
EG_IP=""
while [ -z "$EG_IP" ] && [ $ELAPSED -lt $TIMEOUT ]; do
  echo -e "Waiting for EG_IP... (${ELAPSED}s/${TIMEOUT}s)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  EG_IP=$(kubectl get gateway -n ingress2eg nginx -o jsonpath='{.status.addresses[0].value}' 2>/dev/null || echo "")
done

if [ -z "$EG_IP" ]; then
  echo -e "❌ Error: Timeout waiting for EG_IP"
  exit 1
fi

echo -e ""
echo -e "🚀 INGRESS_IP: ${INGRESS_IP}"
echo -e "🚀 EG_IP: ${EG_IP}"

echo -e ""
echo -e "===================================="
echo -e "NGINX Ingress Upstream VHost Test"
echo -e "===================================="

# Test: Upstream VHost (Host header override)
echo -e ""
echo -e "Testing Upstream VHost (Host header override)..."
RESOLVE_VHOST="--resolve vhost.example.com:80:${INGRESS_IP}"
RESPONSE_VHOST=$(curl -s ${RESOLVE_VHOST} http://vhost.example.com/)
HOST_HEADER=$(echo "$RESPONSE_VHOST" | grep -o '"host":"[^"]*"' | cut -d'"' -f4)

echo -e "  Host header received by backend: ${HOST_HEADER}"

if [ "$HOST_HEADER" = "internal.backend.svc.cluster.local" ]; then
  echo -e "  ✅ Upstream VHost working"
else
  echo -e "  ❌ Upstream VHost failed (expected internal.backend.svc.cluster.local, got ${HOST_HEADER})"
fi

echo -e ""
echo -e "===================================="
echo -e "Envoy Gateway Upstream VHost Test"
echo -e "===================================="

# Test: Upstream VHost (Host header override)
echo -e ""
echo -e "Testing Upstream VHost (Host header override)..."
RESOLVE_VHOST="--resolve vhost.example.com:80:${EG_IP}"
RESPONSE_VHOST=$(curl -s ${RESOLVE_VHOST} http://vhost.example.com/)
HOST_HEADER=$(echo "$RESPONSE_VHOST" | grep -o '"host":"[^"]*"' | cut -d'"' -f4)

echo -e "  Host header received by backend: ${HOST_HEADER}"

if [ "$HOST_HEADER" = "internal.backend.svc.cluster.local" ]; then
  echo -e "  ✅ Upstream VHost working"
else
  echo -e "  ❌ Upstream VHost failed (expected internal.backend.svc.cluster.local, got ${HOST_HEADER})"
fi

echo -e ""
