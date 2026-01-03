#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/session-affinity.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/session-affinity.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/session-affinity.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/session-affinity.yaml

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
  INGRESS_IP=$(kubectl get ingress -n ingress2eg session-affinity -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
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
echo -e "====================================="
echo -e "NGINX Ingress Session Affinity Test"
echo -e "====================================="

RESOLVE="--resolve session.example.com:80:${INGRESS_IP}"

# Test 1: Get session cookie
echo -e ""
echo -e "Test 1: Getting session cookie..."
RESPONSE=$(curl -s ${RESOLVE} -i http://session.example.com/)
COOKIE=$(echo "$RESPONSE" | grep -i "Set-Cookie:" | grep "test-cookie" | head -1)

echo -e "  Set-Cookie: ${COOKIE}"

if [[ "$COOKIE" == *"test-cookie"* ]]; then
  echo -e "  ✅ Cookie name is correct (test-cookie)"
else
  echo -e "  ❌ Cookie name is incorrect (expected test-cookie)"
fi

if [[ "$COOKIE" == *"Max-Age=3600"* ]]; then
  echo -e "  ✅ Max-Age is correct (3600)"
else
  echo -e "  ❌ Max-Age is incorrect (expected 3600)"
fi

if [[ "$COOKIE" == *"SameSite=Lax"* ]]; then
  echo -e "  ✅ SameSite is correct (Lax)"
else
  echo -e "  ❌ SameSite is incorrect (expected Lax)"
fi

# Extract cookie value
COOKIE_VALUE=$(echo "$COOKIE" | sed -n 's/.*test-cookie=\([^;]*\).*/\1/p' | tr -d '\r')

# Test 2: Verify session persistence
echo -e ""
echo -e "Test 2: Verifying session persistence..."

# Make first request with cookie and get hostname
FIRST_HOST=$(curl -s ${RESOLVE} -b "test-cookie=${COOKIE_VALUE}" http://session.example.com/ | grep -o '"HOSTNAME":"[^"]*"' | cut -d'"' -f4)
echo -e "  First request hostname: ${FIRST_HOST}"

# Make 5 more requests with the same cookie
SAME_HOST=0
for i in {1..5}; do
  HOST=$(curl -s ${RESOLVE} -b "test-cookie=${COOKIE_VALUE}" http://session.example.com/ | grep -o '"HOSTNAME":"[^"]*"' | cut -d'"' -f4)
  if [ "$HOST" = "$FIRST_HOST" ]; then
    SAME_HOST=$((SAME_HOST + 1))
  fi
done

echo -e "  Same host count: ${SAME_HOST}/5"

if [ $SAME_HOST -eq 5 ]; then
  echo -e "  ✅ Session affinity working (all requests to same backend)"
else
  echo -e "  ❌ Session affinity not working (requests distributed)"
fi

echo -e ""
echo -e "====================================="
echo -e "Envoy Gateway Session Affinity Test"
echo -e "====================================="

RESOLVE="--resolve session.example.com:80:${EG_IP}"

# Test 1: Get session cookie
echo -e ""
echo -e "Test 1: Getting session cookie..."
RESPONSE=$(curl -s ${RESOLVE} -i http://session.example.com/)
COOKIE=$(echo "$RESPONSE" | grep -i "Set-Cookie:" | grep "test-cookie" | head -1)

echo -e "  Set-Cookie: ${COOKIE}"

if [[ "$COOKIE" == *"test-cookie"* ]]; then
  echo -e "  ✅ Cookie name is correct (test-cookie)"
else
  echo -e "  ❌ Cookie name is incorrect (expected test-cookie)"
fi

if [[ "$COOKIE" == *"Max-Age=3600"* ]]; then
  echo -e "  ✅ Max-Age is correct (3600)"
else
  echo -e "  ❌ Max-Age is incorrect (expected 3600)"
fi

if [[ "$COOKIE" == *"SameSite=Lax"* ]]; then
  echo -e "  ✅ SameSite is correct (Lax)"
else
  echo -e "  ❌ SameSite is incorrect (expected Lax)"
fi

# Extract cookie value
COOKIE_VALUE=$(echo "$COOKIE" | sed -n 's/.*test-cookie=\([^;]*\).*/\1/p' | tr -d '\r')

# Test 2: Verify session persistence
echo -e ""
echo -e "Test 2: Verifying session persistence..."

# Make first request with cookie and get hostname
FIRST_HOST=$(curl -s ${RESOLVE} -b "test-cookie=${COOKIE_VALUE}" http://session.example.com/ | grep -o '"HOSTNAME":"[^"]*"' | cut -d'"' -f4)
echo -e "  First request hostname: ${FIRST_HOST}"

# Make 5 more requests with the same cookie
SAME_HOST=0
for i in {1..5}; do
  HOST=$(curl -s ${RESOLVE} -b "test-cookie=${COOKIE_VALUE}" http://session.example.com/ | grep -o '"HOSTNAME":"[^"]*"' | cut -d'"' -f4)
  if [ "$HOST" = "$FIRST_HOST" ]; then
    SAME_HOST=$((SAME_HOST + 1))
  fi
done

echo -e "  Same host count: ${SAME_HOST}/5"

if [ $SAME_HOST -eq 5 ]; then
  echo -e "  ✅ Session affinity working (all requests to same backend)"
else
  echo -e "  ❌ Session affinity not working (requests distributed)"
fi

echo -e ""
