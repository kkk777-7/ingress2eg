#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/proxy-connect-timeout.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/proxy-connect-timeout.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/proxy-connect-timeout.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/proxy-connect-timeout.yaml

echo -e ""
echo -e "Waiting for IPs..."
echo -e ""

# check for INGRESS_IP and EG_IP
INGRESS_IP=$(kubectl get ingress -n ingress2eg proxy-timeout -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
EG_IP=$(kubectl get gateway -n ingress2eg nginx -o jsonpath='{.status.addresses[0].value}' 2>/dev/null)

# wait for INGRESS_IP with timeout
TIMEOUT=300  # 5 minutes
ELAPSED=0
while [ -z "$INGRESS_IP" ] && [ $ELAPSED -lt $TIMEOUT ]; do
  echo -e "Waiting for INGRESS_IP... (${ELAPSED}s/${TIMEOUT}s)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  INGRESS_IP=$(kubectl get ingress -n ingress2eg proxy-timeout -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
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
echo -e "NGINX Ingress Proxy Timeout Test"
echo -e "===================================="

INGRESS_RESOLVE="--resolve timeout.example.com:80:${INGRESS_IP}"

# test normal endpoint (should succeed)
echo -e ""
echo -e "Testing normal endpoint..."
INGRESS_NORMAL=$(curl -s ${INGRESS_RESOLVE} -o /dev/null -w "%{http_code}" --max-time 10 http://timeout.example.com/normal)

# test timeout endpoint (should timeout with 504)
echo -e "Testing timeout endpoint..."
INGRESS_TIMEOUT_RESPONSE=$(curl -s ${INGRESS_RESOLVE} -o /dev/null -w "%{http_code}|%{time_total}" --max-time 10 http://timeout.example.com/timeout)
INGRESS_TIMEOUT=$(echo $INGRESS_TIMEOUT_RESPONSE | cut -d'|' -f1)
INGRESS_TIMEOUT_TIME=$(echo $INGRESS_TIMEOUT_RESPONSE | cut -d'|' -f2)

# nginx ingress "proxy-stream-next-upstream-tries" default is 3, so INGRESS_TIMEOUT_TIME should be 6+ seconds.
echo -e ""
echo -e "Results:"
echo -e "  Normal endpoint: ${INGRESS_NORMAL}"
echo -e "  Timeout endpoint: ${INGRESS_TIMEOUT} (${INGRESS_TIMEOUT_TIME}s)"
echo -e ""

if [ "$INGRESS_NORMAL" = "200" ] && [ "$INGRESS_TIMEOUT" = "504" ]; then
  echo -e "✅ NGINX Ingress: Proxy connect timeout working correctly!"
  echo -e ""
else
  echo -e "❌ NGINX Ingress: Proxy connect timeout test failed"
  echo -e ""
fi

echo -e "===================================="
echo -e "Envoy Gateway Proxy Timeout Test"
echo -e "===================================="

EG_RESOLVE="--resolve timeout.example.com:80:${EG_IP}"

# test normal endpoint (should succeed)
echo -e ""
echo -e "Testing normal endpoint..."
EG_NORMAL=$(curl -s ${EG_RESOLVE} -o /dev/null -w "%{http_code}" http://timeout.example.com/normal)

# test timeout endpoint (should timeout with 503)
echo -e "Testing timeout endpoint..."
EG_TIMEOUT_RESPONSE=$(curl -s ${EG_RESOLVE} -o /dev/null -w "%{http_code}|%{time_total}" http://timeout.example.com/timeout)
EG_TIMEOUT=$(echo $EG_TIMEOUT_RESPONSE | cut -d'|' -f1)
EG_TIMEOUT_TIME=$(echo $EG_TIMEOUT_RESPONSE | cut -d'|' -f2)

echo -e ""
echo -e "Results:"
echo -e "  Normal endpoint: ${EG_NORMAL}"
echo -e "  Timeout endpoint: ${EG_TIMEOUT} (${EG_TIMEOUT_TIME}s)"
echo -e ""

if [ "$EG_NORMAL" = "200" ] && [ "$EG_TIMEOUT" = "503" ]; then
  echo -e "✅ Envoy Gateway: Proxy connect timeout working correctly!"
  echo -e ""
else
  echo -e "❌ Envoy Gateway: Proxy connect timeout test failed"
  echo -e ""
fi

echo -e ""
