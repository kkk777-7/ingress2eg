#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/ip-access-control.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/ip-access-control.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/ip-access-control.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/ip-access-control.yaml

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
  INGRESS_IP=$(kubectl get ingress -n ingress2eg whitelist-allow -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
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
echo -e "NGINX Ingress IP Access Control Tests"
echo -e "===================================="

# Test 1: Whitelist Allow (0.0.0.0/0 - should allow)
echo -e ""
echo -e "Test 1: Whitelist Allow (0.0.0.0/0)"
RESOLVE_ALLOW="--resolve whitelist-allow.example.com:80:${INGRESS_IP}"
STATUS_ALLOW=$(curl -s ${RESOLVE_ALLOW} -o /dev/null -w "%{http_code}" http://whitelist-allow.example.com/)

echo -e "  Status Code: ${STATUS_ALLOW}"

if [ "$STATUS_ALLOW" = "200" ]; then
  echo -e "  ✅ Whitelist allow working (0.0.0.0/0 permits all)"
else
  echo -e "  ❌ Whitelist allow failed (expected 200)"
fi

# Test 2: Whitelist Deny (192.0.2.0/24 - should deny)
echo -e ""
echo -e "Test 2: Whitelist Deny (192.0.2.0/24 only)"
RESOLVE_DENY="--resolve whitelist-deny.example.com:80:${INGRESS_IP}"
STATUS_DENY=$(curl -s ${RESOLVE_DENY} -o /dev/null -w "%{http_code}" http://whitelist-deny.example.com/)

echo -e "  Status Code: ${STATUS_DENY}"

if [ "$STATUS_DENY" = "403" ]; then
  echo -e "  ✅ Whitelist deny working (IP not in whitelist)"
else
  echo -e "  ❌ Whitelist deny failed (expected 403, got ${STATUS_DENY})"
fi

# Test 3: Denylist Block (192.0.2.0/24 - should allow)
echo -e ""
echo -e "Test 3: Denylist (blocks 192.0.2.0/24)"
RESOLVE_BLOCK="--resolve denylist-block.example.com:80:${INGRESS_IP}"
STATUS_BLOCK=$(curl -s ${RESOLVE_BLOCK} -o /dev/null -w "%{http_code}" http://denylist-block.example.com/)

echo -e "  Status Code: ${STATUS_BLOCK}"

if [ "$STATUS_BLOCK" = "200" ]; then
  echo -e "  ✅ Denylist working (IP not in denylist)"
else
  echo -e "  ❌ Denylist failed (expected 200)"
fi

echo -e ""
echo -e "===================================="
echo -e "Envoy Gateway IP Access Control Tests"
echo -e "===================================="

# Test 1: Whitelist Allow (0.0.0.0/0 - should allow)
echo -e ""
echo -e "Test 1: Whitelist Allow (0.0.0.0/0)"
RESOLVE_ALLOW="--resolve whitelist-allow.example.com:80:${EG_IP}"
STATUS_ALLOW=$(curl -s ${RESOLVE_ALLOW} -o /dev/null -w "%{http_code}" http://whitelist-allow.example.com/)

echo -e "  Status Code: ${STATUS_ALLOW}"

if [ "$STATUS_ALLOW" = "200" ]; then
  echo -e "  ✅ Whitelist allow working (0.0.0.0/0 permits all)"
else
  echo -e "  ❌ Whitelist allow failed (expected 200)"
fi

# Test 2: Whitelist Deny (192.0.2.0/24 - should deny)
echo -e ""
echo -e "Test 2: Whitelist Deny (192.0.2.0/24 only)"
RESOLVE_DENY="--resolve whitelist-deny.example.com:80:${EG_IP}"
STATUS_DENY=$(curl -s ${RESOLVE_DENY} -o /dev/null -w "%{http_code}" http://whitelist-deny.example.com/)

echo -e "  Status Code: ${STATUS_DENY}"

if [ "$STATUS_DENY" = "403" ]; then
  echo -e "  ✅ Whitelist deny working (IP not in whitelist)"
else
  echo -e "  ❌ Whitelist deny failed (expected 403, got ${STATUS_DENY})"
fi

# Test 3: Denylist Block (192.0.2.0/24 - should allow)
echo -e ""
echo -e "Test 3: Denylist (blocks 192.0.2.0/24)"
RESOLVE_BLOCK="--resolve denylist-block.example.com:80:${EG_IP}"
STATUS_BLOCK=$(curl -s ${RESOLVE_BLOCK} -o /dev/null -w "%{http_code}" http://denylist-block.example.com/)

echo -e "  Status Code: ${STATUS_BLOCK}"

if [ "$STATUS_BLOCK" = "200" ]; then
  echo -e "  ✅ Denylist working (IP not in denylist)"
else
  echo -e "  ❌ Denylist failed (expected 200)"
fi

echo -e ""
