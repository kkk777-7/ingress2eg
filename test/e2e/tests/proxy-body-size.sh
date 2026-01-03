#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/proxy-body-size.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/proxy-body-size.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/proxy-body-size.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/proxy-body-size.yaml

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
  INGRESS_IP=$(kubectl get ingress -n ingress2eg proxy-body-size -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
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
echo -e "NGINX Ingress Proxy Body Size Test"
echo -e "===================================="

RESOLVE="--resolve body-size.example.com:80:${INGRESS_IP}"

# Test 1: Small body (should succeed)
echo -e ""
echo -e "Test 1: Small body (500 bytes, limit 1k)"
SMALL_BODY=$(head -c 500 /dev/urandom | base64)
STATUS_SMALL=$(curl -s ${RESOLVE} -X POST -d "$SMALL_BODY" -o /dev/null -w "%{http_code}" http://body-size.example.com/)

echo -e "  Status Code: ${STATUS_SMALL}"

if [ "$STATUS_SMALL" = "200" ]; then
  echo -e "  ✅ Small body accepted"
else
  echo -e "  ❌ Small body rejected (expected 200, got ${STATUS_SMALL})"
fi

# Test 2: Large body (should fail)
echo -e ""
echo -e "Test 2: Large body (2048 bytes, limit 1k)"
LARGE_BODY=$(head -c 2048 /dev/urandom | base64)
STATUS_LARGE=$(curl -s ${RESOLVE} -X POST -d "$LARGE_BODY" -o /dev/null -w "%{http_code}" http://body-size.example.com/)

echo -e "  Status Code: ${STATUS_LARGE}"

if [ "$STATUS_LARGE" = "413" ]; then
  echo -e "  ✅ Large body rejected (413 Request Entity Too Large)"
else
  echo -e "  ❌ Large body not rejected properly (expected 413, got ${STATUS_LARGE})"
fi

echo -e ""
echo -e "===================================="
echo -e "Envoy Gateway Proxy Body Size Test"
echo -e "===================================="

RESOLVE="--resolve body-size.example.com:80:${EG_IP}"

# Test 1: Small body (should succeed)
echo -e ""
echo -e "Test 1: Small body (500 bytes, limit 1k)"
SMALL_BODY=$(head -c 500 /dev/urandom | base64)
STATUS_SMALL=$(curl -s ${RESOLVE} -X POST -d "$SMALL_BODY" -o /dev/null -w "%{http_code}" http://body-size.example.com/)

echo -e "  Status Code: ${STATUS_SMALL}"

if [ "$STATUS_SMALL" = "200" ]; then
  echo -e "  ✅ Small body accepted"
else
  echo -e "  ❌ Small body rejected (expected 200, got ${STATUS_SMALL})"
fi

# Test 2: Large body (should fail)
echo -e ""
echo -e "Test 2: Large body (2048 bytes, limit 1k)"
LARGE_BODY=$(head -c 2048 /dev/urandom | base64)
STATUS_LARGE=$(curl -s ${RESOLVE} -X POST -d "$LARGE_BODY" -o /dev/null -w "%{http_code}" http://body-size.example.com/)

echo -e "  Status Code: ${STATUS_LARGE}"

if [ "$STATUS_LARGE" = "413" ]; then
  echo -e "  ✅ Large body rejected (413 Request Entity Too Large)"
else
  echo -e "  ❌ Large body not rejected properly (expected 413, got ${STATUS_LARGE})"
fi

echo -e ""
