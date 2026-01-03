#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/proxy-next-upstream.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/proxy-next-upstream.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/proxy-next-upstream.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/proxy-next-upstream.yaml

echo -e ""
echo -e "Waiting for IPs..."
echo -e ""

# check for INGRESS_IP and EG_IP
INGRESS_IP=$(kubectl get ingress -n ingress2eg proxy-next-upstream -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
EG_IP=$(kubectl get gateway -n ingress2eg nginx -o jsonpath='{.status.addresses[0].value}' 2>/dev/null)

# wait for INGRESS_IP with timeout
TIMEOUT=300  # 5 minutes
ELAPSED=0
while [ -z "$INGRESS_IP" ] && [ $ELAPSED -lt $TIMEOUT ]; do
  echo -e "Waiting for INGRESS_IP... (${ELAPSED}s/${TIMEOUT}s)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  INGRESS_IP=$(kubectl get ingress -n ingress2eg proxy-next-upstream -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
done

if [ -z "$INGRESS_IP" ]; then
  echo -e "❌ Error: Timeout waiting for INGRESS_IP"
  exit 1
fi

# wait for EG_IP with timeout
ELAPSED=0
while [ -z "$EG_IP" ] && [ $ELAPSED -lt $TIMEOUT ]; do
  echo -e "Waiting for EG_IP... (${ELAPSED}s/${TIMEOUT}s)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  EG_IP=$(kubectl get gateway -n ingress2eg nginx -o jsonpath='{.status.addresses[0].value}' 2>/dev/null)
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
echo -e "NGINX Ingress Proxy Next Upstream Test"
echo -e "===================================="

INGRESS_RESOLVE="--resolve retry.example.com:80:${INGRESS_IP}"

# test retry behavior by checking backend logs
echo -e ""
echo -e "Testing retry behavior (backend returns 503)..."

INGRESS_RESPONSE=$(curl -s ${INGRESS_RESOLVE} -w "\n%{http_code}" http://retry.example.com/test | tail -1)
INGRESS_RETRY_COUNT=$(kubectl logs -l app=error-backend -n ingress2eg --since=5s 2>/dev/null | grep -c "GET /test")

echo -e ""
echo -e "Results:"
echo -e "  Status Code: ${INGRESS_RESPONSE}"
echo -e "  Backend received requests: ${INGRESS_RETRY_COUNT}"
echo -e ""

if [ "$INGRESS_RESPONSE" = "503" ] && [ "$INGRESS_RETRY_COUNT" = 3 ]; then
  echo -e "✅ NGINX Ingress: Retry behavior working"
  echo -e ""
else
  echo -e "❌ NGINX Ingress: Retry behavior may not be working correctly"
  echo -e ""
fi

sleep 5

echo -e "===================================="
echo -e "Envoy Gateway Proxy Next Upstream Test"
echo -e "===================================="

EG_RESOLVE="--resolve retry.example.com:80:${EG_IP}"

# test retry behavior by checking backend logs
echo -e ""
echo -e "Testing retry behavior (backend returns 503)..."

EG_RESPONSE=$(curl -s ${EG_RESOLVE} -w "\n%{http_code}" http://retry.example.com/test | tail -1)
EG_RETRY_COUNT=$(kubectl logs -l app=error-backend -n ingress2eg --since=5s 2>/dev/null | grep -c "GET /test")

echo -e ""
echo -e "Results:"
echo -e "  Status Code: ${EG_RESPONSE}"
echo -e "  Backend received requests: ${EG_RETRY_COUNT}"
echo -e ""

if [ "$EG_RESPONSE" = "503" ] && [ "$EG_RETRY_COUNT" = 3 ]; then
  echo -e "✅ Envoy Gateway: Retry behavior working"
  echo -e ""
else
  echo -e "❌ Envoy Gateway: Retry behavior may not be working correctly"
  echo -e ""
fi

echo -e ""
