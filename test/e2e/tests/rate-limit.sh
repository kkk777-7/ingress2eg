#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/rate-limit.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/rate-limit.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/rate-limit.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/rate-limit.yaml

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
  INGRESS_IP=$(kubectl get ingress -n ingress2eg rate-limit-rps -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
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
echo -e "NGINX Ingress Rate Limit Tests"
echo -e "===================================="

# Test 1: RPS (Requests Per Second) limit
echo -e ""
echo -e "Test 1: RPS Limit (2 requests/second)"
RESOLVE_RPS="--resolve rps.example.com:80:${INGRESS_IP}"

SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0

echo -e "Making 20 rapid requests..."
for i in {1..20}; do
  STATUS=$(curl -s ${RESOLVE_RPS} -o /dev/null -w "%{http_code}" http://rps.example.com/)
  if [ "$STATUS" = "200" ]; then
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
  elif [ "$STATUS" = "503" ]; then
    RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
  fi
  sleep 0.1
done

echo -e ""
echo -e "Results:"
echo -e "  Successful requests (200): ${SUCCESS_COUNT}"
echo -e "  Rate limited requests (503): ${RATE_LIMITED_COUNT}"
echo -e ""

if [ "$RATE_LIMITED_COUNT" -gt 0 ]; then
  echo -e "✅ NGINX Ingress: RPS rate limiting working"
else
  echo -e "❌ NGINX Ingress: RPS rate limiting may not be working"
fi

# Test 2: RPM (Requests Per Minute) limit
echo -e ""
echo -e "Test 2: RPM Limit (10 requests/minute)"
RESOLVE_RPM="--resolve rpm.example.com:80:${INGRESS_IP}"

SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0

echo -e "Making 15 requests..."
for i in {1..15}; do
  STATUS=$(curl -s ${RESOLVE_RPM} -o /dev/null -w "%{http_code}" http://rpm.example.com/)
  if [ "$STATUS" = "200" ]; then
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
  elif [ "$STATUS" = "503" ]; then
    RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
  fi
  sleep 0.5
done

echo -e ""
echo -e "Results:"
echo -e "  Successful requests (200): ${SUCCESS_COUNT}"
echo -e "  Rate limited requests (503): ${RATE_LIMITED_COUNT}"
echo -e ""

if [ "$RATE_LIMITED_COUNT" -gt 0 ]; then
  echo -e "✅ NGINX Ingress: RPM rate limiting working"
else
  echo -e "❌ NGINX Ingress: RPM rate limiting may not be working"
fi

echo -e ""
echo -e "===================================="
echo -e "Envoy Gateway Rate Limit Tests"
echo -e "===================================="

# Test 1: RPS (Requests Per Second) limit
echo -e ""
echo -e "Test 1: RPS Limit (2 requests/second)"
RESOLVE_RPS="--resolve rps.example.com:80:${EG_IP}"

SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0

echo -e "Making 20 rapid requests..."
for i in {1..20}; do
  STATUS=$(curl -s ${RESOLVE_RPS} -o /dev/null -w "%{http_code}" http://rps.example.com/)
  if [ "$STATUS" = "200" ]; then
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
  elif [ "$STATUS" = "429" ]; then
    RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
  fi
  sleep 0.1
done

echo -e ""
echo -e "Results:"
echo -e "  Successful requests (200): ${SUCCESS_COUNT}"
echo -e "  Rate limited requests (429): ${RATE_LIMITED_COUNT}"
echo -e ""

if [ "$RATE_LIMITED_COUNT" -gt 0 ]; then
  echo -e "✅ Envoy Gateway: RPS rate limiting working"
else
  echo -e "❌ Envoy Gateway: RPS rate limiting may not be working"
fi

# Test 2: RPM (Requests Per Minute) limit
echo -e ""
echo -e "Test 2: RPM Limit (10 requests/minute)"
RESOLVE_RPM="--resolve rpm.example.com:80:${EG_IP}"

SUCCESS_COUNT=0
RATE_LIMITED_COUNT=0

echo -e "Making 15 requests..."
for i in {1..15}; do
  STATUS=$(curl -s ${RESOLVE_RPM} -o /dev/null -w "%{http_code}" http://rpm.example.com/)
  if [ "$STATUS" = "200" ]; then
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
  elif [ "$STATUS" = "429" ]; then
    RATE_LIMITED_COUNT=$((RATE_LIMITED_COUNT + 1))
  fi
  sleep 0.5
done

echo -e ""
echo -e "Results:"
echo -e "  Successful requests (200): ${SUCCESS_COUNT}"
echo -e "  Rate limited requests (429): ${RATE_LIMITED_COUNT}"
echo -e ""

if [ "$RATE_LIMITED_COUNT" -gt 0 ]; then
  echo -e "✅ Envoy Gateway: RPM rate limiting working"
else
  echo -e "❌ Envoy Gateway: RPM rate limiting may not be working"
fi

echo -e ""
