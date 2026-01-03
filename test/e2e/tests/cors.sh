#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/cors.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/cors.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/cors.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/cors.yaml

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
  INGRESS_IP=$(kubectl get ingress -n ingress2eg cors -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
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
echo -e "NGINX Ingress CORS Test"
echo -e "====================================="

RESOLVE="--resolve cors.example.com:80:${INGRESS_IP}"

# Test 1: Preflight request
echo -e ""
echo -e "Test 1: CORS preflight request (OPTIONS)..."
RESPONSE=$(curl -s ${RESOLVE} -X OPTIONS -i \
  -H "Origin: https://example.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type,Authorization" \
  http://cors.example.com/)

echo -e "  Response headers:"
echo "$RESPONSE" | grep -i "access-control" || true

ALLOW_ORIGIN=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Origin:" | tr -d '\r' | cut -d' ' -f2-)
ALLOW_METHODS=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Methods:" | tr -d '\r' | cut -d' ' -f2-)
ALLOW_HEADERS=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Headers:" | tr -d '\r' | cut -d' ' -f2-)
MAX_AGE=$(echo "$RESPONSE" | grep -i "Access-Control-Max-Age:" | tr -d '\r' | cut -d' ' -f2-)
ALLOW_CREDENTIALS=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Credentials:" | tr -d '\r' | cut -d' ' -f2-)

echo -e ""
if [[ "$ALLOW_ORIGIN" == *"https://example.com"* ]]; then
  echo -e "  ✅ Access-Control-Allow-Origin is correct"
else
  echo -e "  ❌ Access-Control-Allow-Origin is incorrect (got: ${ALLOW_ORIGIN})"
fi

if [[ "$ALLOW_METHODS" == *"GET"* ]] && [[ "$ALLOW_METHODS" == *"POST"* ]] && [[ "$ALLOW_METHODS" == *"PUT"* ]] && [[ "$ALLOW_METHODS" == *"DELETE"* ]]; then
  echo -e "  ✅ Access-Control-Allow-Methods is correct"
else
  echo -e "  ❌ Access-Control-Allow-Methods is incorrect (got: ${ALLOW_METHODS})"
fi

if [[ "$ALLOW_HEADERS" == *"Content-Type"* ]] && [[ "$ALLOW_HEADERS" == *"Authorization"* ]]; then
  echo -e "  ✅ Access-Control-Allow-Headers is correct"
else
  echo -e "  ❌ Access-Control-Allow-Headers is incorrect (got: ${ALLOW_HEADERS})"
fi

if [[ "$MAX_AGE" == "3600" ]]; then
  echo -e "  ✅ Access-Control-Max-Age is correct (3600)"
else
  echo -e "  ❌ Access-Control-Max-Age is incorrect (expected 3600, got: ${MAX_AGE})"
fi

if [[ "$ALLOW_CREDENTIALS" == "true" ]]; then
  echo -e "  ✅ Access-Control-Allow-Credentials is correct (true)"
else
  echo -e "  ❌ Access-Control-Allow-Credentials is incorrect (expected true, got: ${ALLOW_CREDENTIALS})"
fi

# Test 2: Actual request
echo -e ""
echo -e "Test 2: Actual CORS request (GET)..."
RESPONSE=$(curl -s ${RESOLVE} -i \
  -H "Origin: https://example.com" \
  http://cors.example.com/)

echo -e "  Response headers:"
echo "$RESPONSE" | grep -i "access-control" || true

ALLOW_ORIGIN=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Origin:" | tr -d '\r' | cut -d' ' -f2-)
EXPOSE_HEADERS=$(echo "$RESPONSE" | grep -i "Access-Control-Expose-Headers:" | tr -d '\r' | cut -d' ' -f2-)
ALLOW_CREDENTIALS=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Credentials:" | tr -d '\r' | cut -d' ' -f2-)

echo -e ""
if [[ "$ALLOW_ORIGIN" == *"https://example.com"* ]]; then
  echo -e "  ✅ Access-Control-Allow-Origin is correct"
else
  echo -e "  ❌ Access-Control-Allow-Origin is incorrect (got: ${ALLOW_ORIGIN})"
fi

if [[ "$EXPOSE_HEADERS" == *"Content-Length"* ]] && [[ "$EXPOSE_HEADERS" == *"X-Custom-Response"* ]]; then
  echo -e "  ✅ Access-Control-Expose-Headers is correct"
else
  echo -e "  ❌ Access-Control-Expose-Headers is incorrect (got: ${EXPOSE_HEADERS})"
fi

if [[ "$ALLOW_CREDENTIALS" == "true" ]]; then
  echo -e "  ✅ Access-Control-Allow-Credentials is correct (true)"
else
  echo -e "  ❌ Access-Control-Allow-Credentials is incorrect (expected true, got: ${ALLOW_CREDENTIALS})"
fi

echo -e ""
echo -e "====================================="
echo -e "Envoy Gateway CORS Test"
echo -e "====================================="

RESOLVE="--resolve cors.example.com:80:${EG_IP}"

# Test 1: Preflight request
echo -e ""
echo -e "Test 1: CORS preflight request (OPTIONS)..."
RESPONSE=$(curl -s ${RESOLVE} -X OPTIONS -i \
  -H "Origin: https://example.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type,Authorization" \
  http://cors.example.com/)

echo -e "  Response headers:"
echo "$RESPONSE" | grep -i "access-control" || true

ALLOW_ORIGIN=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Origin:" | tr -d '\r' | cut -d' ' -f2-)
ALLOW_METHODS=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Methods:" | tr -d '\r' | cut -d' ' -f2-)
ALLOW_HEADERS=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Headers:" | tr -d '\r' | cut -d' ' -f2-)
MAX_AGE=$(echo "$RESPONSE" | grep -i "Access-Control-Max-Age:" | tr -d '\r' | cut -d' ' -f2-)
ALLOW_CREDENTIALS=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Credentials:" | tr -d '\r' | cut -d' ' -f2-)

echo -e ""
if [[ "$ALLOW_ORIGIN" == *"https://example.com"* ]]; then
  echo -e "  ✅ Access-Control-Allow-Origin is correct"
else
  echo -e "  ❌ Access-Control-Allow-Origin is incorrect (got: ${ALLOW_ORIGIN})"
fi

if [[ "$ALLOW_METHODS" == *"GET"* ]] && [[ "$ALLOW_METHODS" == *"POST"* ]] && [[ "$ALLOW_METHODS" == *"PUT"* ]] && [[ "$ALLOW_METHODS" == *"DELETE"* ]]; then
  echo -e "  ✅ Access-Control-Allow-Methods is correct"
else
  echo -e "  ❌ Access-Control-Allow-Methods is incorrect (got: ${ALLOW_METHODS})"
fi

if [[ "$ALLOW_HEADERS" == *"Content-Type"* ]] && [[ "$ALLOW_HEADERS" == *"Authorization"* ]]; then
  echo -e "  ✅ Access-Control-Allow-Headers is correct"
else
  echo -e "  ❌ Access-Control-Allow-Headers is incorrect (got: ${ALLOW_HEADERS})"
fi

if [[ "$MAX_AGE" == "3600" ]]; then
  echo -e "  ✅ Access-Control-Max-Age is correct (3600)"
else
  echo -e "  ❌ Access-Control-Max-Age is incorrect (expected 3600, got: ${MAX_AGE})"
fi

if [[ "$ALLOW_CREDENTIALS" == "true" ]]; then
  echo -e "  ✅ Access-Control-Allow-Credentials is correct (true)"
else
  echo -e "  ❌ Access-Control-Allow-Credentials is incorrect (expected true, got: ${ALLOW_CREDENTIALS})"
fi

# Test 2: Actual request
echo -e ""
echo -e "Test 2: Actual CORS request (GET)..."
RESPONSE=$(curl -s ${RESOLVE} -i \
  -H "Origin: https://example.com" \
  http://cors.example.com/)

echo -e "  Response headers:"
echo "$RESPONSE" | grep -i "access-control" || true

ALLOW_ORIGIN=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Origin:" | tr -d '\r' | cut -d' ' -f2-)
EXPOSE_HEADERS=$(echo "$RESPONSE" | grep -i "Access-Control-Expose-Headers:" | tr -d '\r' | cut -d' ' -f2-)
ALLOW_CREDENTIALS=$(echo "$RESPONSE" | grep -i "Access-Control-Allow-Credentials:" | tr -d '\r' | cut -d' ' -f2-)

echo -e ""
if [[ "$ALLOW_ORIGIN" == *"https://example.com"* ]]; then
  echo -e "  ✅ Access-Control-Allow-Origin is correct"
else
  echo -e "  ❌ Access-Control-Allow-Origin is incorrect (got: ${ALLOW_ORIGIN})"
fi

if [[ "$EXPOSE_HEADERS" == *"Content-Length"* ]] && [[ "$EXPOSE_HEADERS" == *"X-Custom-Response"* ]]; then
  echo -e "  ✅ Access-Control-Expose-Headers is correct"
else
  echo -e "  ❌ Access-Control-Expose-Headers is incorrect (got: ${EXPOSE_HEADERS})"
fi

if [[ "$ALLOW_CREDENTIALS" == "true" ]]; then
  echo -e "  ✅ Access-Control-Allow-Credentials is correct (true)"
else
  echo -e "  ❌ Access-Control-Allow-Credentials is incorrect (expected true, got: ${ALLOW_CREDENTIALS})"
fi

echo -e ""
