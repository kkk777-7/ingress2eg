#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/canary.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/canary.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/canary.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/canary.yaml

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
  INGRESS_IP=$(kubectl get ingress -n ingress2eg canary-main -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
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
echo -e "NGINX Ingress Canary Tests"
echo -e "===================================="

# Test 1: Canary by header
echo -e ""
echo -e "Test 1: Canary by header (X-Canary: always)"
RESOLVE_CANARY="--resolve canary.example.com:80:${INGRESS_IP}"

# Request with canary header (should go to canary)
RESPONSE_CANARY=$(curl -s ${RESOLVE_CANARY} -H "X-Canary: always" http://canary.example.com/)
SERVICE_CANARY=$(echo "$RESPONSE_CANARY" | grep -o '"HOSTNAME":"[^"]*"' | cut -d'"' -f4)

echo -e "  Service with header: ${SERVICE_CANARY}"

if [[ "$SERVICE_CANARY" == *"canary"* ]]; then
  echo -e "  ✅ Canary by header working"
else
  echo -e "  ❌ Canary by header failed (expected canary service)"
fi

# Test 2: Canary by weight
echo -e ""
echo -e "Test 2: Canary by weight (30% canary)"
RESOLVE_WEIGHT="--resolve canary-weight.example.com:80:${INGRESS_IP}"

MAIN_COUNT=0
CANARY_COUNT=0
TOTAL_REQUESTS=50

echo -e "  Making ${TOTAL_REQUESTS} requests to measure traffic split..."

for i in $(seq 1 $TOTAL_REQUESTS); do
  RESPONSE=$(curl -s ${RESOLVE_WEIGHT} http://canary-weight.example.com/)
  SERVICE=$(echo "$RESPONSE" | grep -o '"HOSTNAME":"[^"]*"' | cut -d'"' -f4)

  if [[ "$SERVICE" == *"canary"* ]]; then
    CANARY_COUNT=$((CANARY_COUNT + 1))
  else
    MAIN_COUNT=$((MAIN_COUNT + 1))
  fi

  sleep 0.1
done

MAIN_PERCENT=$((MAIN_COUNT * 100 / TOTAL_REQUESTS))
CANARY_PERCENT=$((CANARY_COUNT * 100 / TOTAL_REQUESTS))

echo -e ""
echo -e "  Results (${TOTAL_REQUESTS} requests):"
echo -e "    Main service: ${MAIN_COUNT} (${MAIN_PERCENT}%)"
echo -e "    Canary service: ${CANARY_COUNT} (${CANARY_PERCENT}%)"
echo -e ""

# Allow 10% margin of error (target 30%, acceptable 20-40%)
if [ "$CANARY_PERCENT" -ge 20 ] && [ "$CANARY_PERCENT" -le 40 ]; then
  echo -e "  ✅ Canary weight working (${CANARY_PERCENT}% canary traffic)"
else
  echo -e "  ⚠️  Canary weight may not be accurate (expected ~30%, got ${CANARY_PERCENT}%)"
fi

echo -e ""
echo -e "===================================="
echo -e "Envoy Gateway Canary Tests"
echo -e "===================================="

# Test 1: Canary by header
echo -e ""
echo -e "Test 1: Canary by header (X-Canary: always)"
RESOLVE_CANARY="--resolve canary.example.com:80:${EG_IP}"

# Request with canary header (should go to canary)
RESPONSE_CANARY=$(curl -s ${RESOLVE_CANARY} -H "X-Canary: always" http://canary.example.com/)
SERVICE_CANARY=$(echo "$RESPONSE_CANARY" | grep -o '"HOSTNAME":"[^"]*"' | cut -d'"' -f4)

echo -e "  Service with header: ${SERVICE_CANARY}"

if [[ "$SERVICE_CANARY" == *"canary"* ]]; then
  echo -e "  ✅ Canary by header working"
else
  echo -e "  ❌ Canary by header failed (expected canary service)"
fi

# Test 2: Canary by weight
echo -e ""
echo -e "Test 2: Canary by weight (30% canary)"
RESOLVE_WEIGHT="--resolve canary-weight.example.com:80:${EG_IP}"

MAIN_COUNT=0
CANARY_COUNT=0
TOTAL_REQUESTS=50

echo -e "  Making ${TOTAL_REQUESTS} requests to measure traffic split..."

for i in $(seq 1 $TOTAL_REQUESTS); do
  RESPONSE=$(curl -s ${RESOLVE_WEIGHT} http://canary-weight.example.com/)
  SERVICE=$(echo "$RESPONSE" | grep -o '"HOSTNAME":"[^"]*"' | cut -d'"' -f4)

  if [[ "$SERVICE" == *"canary"* ]]; then
    CANARY_COUNT=$((CANARY_COUNT + 1))
  else
    MAIN_COUNT=$((MAIN_COUNT + 1))
  fi

  sleep 0.1
done

MAIN_PERCENT=$((MAIN_COUNT * 100 / TOTAL_REQUESTS))
CANARY_PERCENT=$((CANARY_COUNT * 100 / TOTAL_REQUESTS))

echo -e ""
echo -e "  Results (${TOTAL_REQUESTS} requests):"
echo -e "    Main service: ${MAIN_COUNT} (${MAIN_PERCENT}%)"
echo -e "    Canary service: ${CANARY_COUNT} (${CANARY_PERCENT}%)"
echo -e ""

# Allow 10% margin of error (target 30%, acceptable 20-40%)
if [ "$CANARY_PERCENT" -ge 20 ] && [ "$CANARY_PERCENT" -le 40 ]; then
  echo -e "  ✅ Canary weight working (${CANARY_PERCENT}% canary traffic)"
else
  echo -e "  ⚠️  Canary weight may not be accurate (expected ~30%, got ${CANARY_PERCENT}%)"
fi

echo -e ""
