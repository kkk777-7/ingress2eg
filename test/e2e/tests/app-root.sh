#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/app-root.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/app-root.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/app-root.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/app-root.yaml

echo -e ""
echo -e "Waiting for IPs..."
echo -e ""

# check for INGRESS_IP and EG_IP
INGRESS_IP=$(kubectl get ingress -n ingress2eg app-root -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
EG_IP=$(kubectl get gateway -n ingress2eg nginx -o jsonpath='{.status.addresses[0].value}' 2>/dev/null)

# wait for INGRESS_IP with timeout
TIMEOUT=300  # 5 minutes
ELAPSED=0
while [ -z "$INGRESS_IP" ] && [ $ELAPSED -lt $TIMEOUT ]; do
  echo -e "Waiting for INGRESS_IP... (${ELAPSED}s/${TIMEOUT}s)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  INGRESS_IP=$(kubectl get ingress -n ingress2eg app-root -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
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
echo -e "NGINX Ingress App Root Test"
echo -e "===================================="

INGRESS_RESOLVE="--resolve approot.example.com:80:${INGRESS_IP}"

# test root path redirect
echo -e ""
echo -e "Testing root path redirect (/ -> /app)..."
INGRESS_REDIRECT=$(curl -s ${INGRESS_RESOLVE} -w "\n%{redirect_url}|%{http_code}" http://approot.example.com/ | tail -1)
INGRESS_REDIRECT_URL=$(echo "$INGRESS_REDIRECT" | cut -d'|' -f1)
INGRESS_REDIRECT_CODE=$(echo "$INGRESS_REDIRECT" | cut -d'|' -f2)

echo -e ""
echo -e "Results:"
echo -e "  Redirect URL: ${INGRESS_REDIRECT_URL}"
echo -e "  Redirect Status: ${INGRESS_REDIRECT_CODE}"
echo -e ""

# test non-root path (should not redirect)
echo -e "Testing non-root path (/test -> should not redirect)..."
INGRESS_NORMAL=$(curl -s ${INGRESS_RESOLVE} -w "\n%{redirect_url}|%{http_code}" http://approot.example.com/test | tail -1)
INGRESS_NORMAL_URL=$(echo "$INGRESS_NORMAL" | cut -d'|' -f1)
INGRESS_NORMAL_CODE=$(echo "$INGRESS_NORMAL" | cut -d'|' -f2)

echo -e ""
echo -e "Results:"
echo -e "  Redirect URL: ${INGRESS_NORMAL_URL}"
echo -e "  Status Code: ${INGRESS_NORMAL_CODE}"
echo -e ""

if [ "$INGRESS_REDIRECT_CODE" = "302" ] && [ "$INGRESS_REDIRECT_URL" = "http://approot.example.com/app" ] && \
   [ "$INGRESS_NORMAL_CODE" = "200" ] && [ -z "$INGRESS_NORMAL_URL" ]; then
  echo -e "✅ NGINX Ingress: App root working correctly!"
  echo -e ""
else
  echo -e "❌ NGINX Ingress: App root test failed"
  echo -e ""
fi

echo -e "===================================="
echo -e "Envoy Gateway App Root Test"
echo -e "===================================="

EG_RESOLVE="--resolve approot.example.com:80:${EG_IP}"

# test root path redirect
echo -e ""
echo -e "Testing root path redirect (/ -> /app)..."
EG_REDIRECT=$(curl -s ${EG_RESOLVE} -w "\n%{redirect_url}|%{http_code}" http://approot.example.com/ | tail -1)
EG_REDIRECT_URL=$(echo "$EG_REDIRECT" | cut -d'|' -f1)
EG_REDIRECT_CODE=$(echo "$EG_REDIRECT" | cut -d'|' -f2)

echo -e ""
echo -e "Results:"
echo -e "  Redirect URL: ${EG_REDIRECT_URL}"
echo -e "  Redirect Status: ${EG_REDIRECT_CODE}"
echo -e ""

# test non-root path (should not redirect)
echo -e "Testing non-root path (/test -> should not redirect)..."
EG_NORMAL=$(curl -s ${EG_RESOLVE} -w "\n%{redirect_url}|%{http_code}" http://approot.example.com/test | tail -1)
EG_NORMAL_URL=$(echo "$EG_NORMAL" | cut -d'|' -f1)
EG_NORMAL_CODE=$(echo "$EG_NORMAL" | cut -d'|' -f2)

echo -e ""
echo -e "Results:"
echo -e "  Redirect URL: ${EG_NORMAL_URL}"
echo -e "  Status Code: ${EG_NORMAL_CODE}"
echo -e ""

if [ "$EG_REDIRECT_CODE" = "302" ] && [ "$EG_REDIRECT_URL" = "http://approot.example.com/app" ] && \
   [ "$EG_NORMAL_CODE" = "200" ] && [ -z "$EG_NORMAL_URL" ]; then
  echo -e "✅ Envoy Gateway: App root working correctly!"
  echo -e ""
else
  echo -e "❌ Envoy Gateway: App root test failed"
  echo -e ""
fi

echo -e ""
