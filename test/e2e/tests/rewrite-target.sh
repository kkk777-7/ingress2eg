#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/rewrite-target.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/rewrite-target.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/rewrite-target.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/rewrite-target.yaml

echo -e ""
echo -e "Waiting for IPs..."
echo -e ""

# check for INGRESS_IP and EG_IP
INGRESS_IP=$(kubectl get ingress -n ingress2eg rewrite-target -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
EG_IP=$(kubectl get gateway -n ingress2eg nginx -o jsonpath='{.status.addresses[0].value}' 2>/dev/null)

# wait for INGRESS_IP with timeout
TIMEOUT=300  # 5 minutes
ELAPSED=0
while [ -z "$INGRESS_IP" ] && [ $ELAPSED -lt $TIMEOUT ]; do
  echo -e "Waiting for INGRESS_IP... (${ELAPSED}s/${TIMEOUT}s)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  INGRESS_IP=$(kubectl get ingress -n ingress2eg rewrite-target -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
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
echo -e "NGINX Ingress Rewrite Target Test"
echo -e "===================================="

INGRESS_RESOLVE="--resolve rewrite.example.com:80:${INGRESS_IP}"

# test rewritten path
echo -e ""
echo -e "Testing rewritten path (/test -> /rewritten/user1)..."
INGRESS_REWRITE_RESPONSE=$(curl -s ${INGRESS_RESOLVE} http://rewrite.example.com/test/user1)
INGRESS_REWRITE_PATH=$(echo "$INGRESS_REWRITE_RESPONSE" | grep -o '"originalUrl":"[^"]*"' | cut -d'"' -f4)
INGRESS_REWRITE_STATUS=$(curl -s ${INGRESS_RESOLVE} -o /dev/null -w "%{http_code}" http://rewrite.example.com/test/user1)

echo -e ""
echo -e "Results:"
echo -e "  Rewritten path: ${INGRESS_REWRITE_PATH} (status: ${INGRESS_REWRITE_STATUS})"
echo -e ""

if [ "$INGRESS_REWRITE_STATUS" = "200" ] && [ "$INGRESS_REWRITE_PATH" = "/rewritten/user1" ]; then
  echo -e "✅ NGINX Ingress: Rewrite target working correctly!"
  echo -e ""
else
  echo -e "❌ NGINX Ingress: Rewrite target test failed"
  echo -e ""
fi

echo -e "===================================="
echo -e "Envoy Gateway Rewrite Target Test"
echo -e "===================================="

EG_RESOLVE="--resolve rewrite.example.com:80:${EG_IP}"

# test rewritten path
echo -e ""
echo -e "Testing rewritten path (/test -> /rewritten/user1)..."
EG_REWRITE_RESPONSE=$(curl -s ${EG_RESOLVE} http://rewrite.example.com/test/user1)
EG_REWRITE_PATH=$(echo "$EG_REWRITE_RESPONSE" | grep -o '"originalUrl":"[^"]*"' | cut -d'"' -f4)
EG_REWRITE_STATUS=$(curl -s ${EG_RESOLVE} -o /dev/null -w "%{http_code}" http://rewrite.example.com/test/user1)

echo -e ""
echo -e "Results:"
echo -e "  Rewritten path: ${EG_REWRITE_PATH} (status: ${EG_REWRITE_STATUS})"
echo -e ""

if [ "$EG_REWRITE_STATUS" = "200" ] && [ "$EG_REWRITE_PATH" = "/rewritten/user1" ]; then
  echo -e "✅ Envoy Gateway: Rewrite target working correctly!"
  echo -e ""
else
  echo -e "❌ Envoy Gateway: Rewrite target test failed"
  echo -e ""
fi

echo -e ""
