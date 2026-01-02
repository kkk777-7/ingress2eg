#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/ssl-passthrough.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/ssl-passthrough.yaml 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/ssl-passthrough.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/ssl-passthrough.yaml

echo -e ""
echo -e "Waiting for IPs..."
echo -e ""

# check for INGRESS_IP and EG_IP
INGRESS_IP=$(kubectl get ingress -n ingress2eg ssl-passthrough -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
EG_IP=$(kubectl get gateway -n ingress2eg nginx -o jsonpath='{.status.addresses[0].value}' 2>/dev/null)

# wait for INGRESS_IP with timeout
TIMEOUT=300  # 5 minutes
ELAPSED=0
while [ -z "$INGRESS_IP" ] && [ $ELAPSED -lt $TIMEOUT ]; do
  echo -e "Waiting for INGRESS_IP... (${ELAPSED}s/${TIMEOUT}s)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  INGRESS_IP=$(kubectl get ingress -n ingress2eg ssl-passthrough -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
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
echo -e "NGINX Ingress SSL Passthrough Test"
echo -e "===================================="

# test TLS connection and get certificate subject
echo -e ""
echo -e "Testing TLS connection..."
INGRESS_CERT_SUBJECT=$(echo "Q" | openssl s_client -connect ${INGRESS_IP}:443 \
  -servername passthrough.example.com 2>/dev/null | \
  openssl x509 -noout -subject 2>/dev/null || echo "")

# test HTTPS request
INGRESS_HTTPS=$(curl -ks --resolve passthrough.example.com:443:${INGRESS_IP} \
  -o /dev/null -w "%{http_code}" https://passthrough.example.com/)

echo -e ""
echo -e "Results:"
echo -e "  Certificate Subject: ${INGRESS_CERT_SUBJECT}"
echo -e "  HTTPS Status: ${INGRESS_HTTPS}"
echo -e ""

if [ -n "$INGRESS_CERT_SUBJECT" ] && [ "$INGRESS_HTTPS" = "200" ]; then
  echo -e "✅ NGINX Ingress: SSL passthrough working!"
  echo -e ""
else
  echo -e "❌ NGINX Ingress: SSL passthrough test failed"
  echo -e ""
fi

echo -e "===================================="
echo -e "Envoy Gateway SSL Passthrough Test"
echo -e "===================================="

# test TLS connection and get certificate subject
echo -e ""
echo -e "Testing TLS connection..."
EG_CERT_SUBJECT=$(echo "Q" | openssl s_client -connect ${EG_IP}:443 \
  -servername passthrough.example.com 2>/dev/null | \
  openssl x509 -noout -subject 2>/dev/null || echo "")

# test HTTPS request
EG_HTTPS=$(curl -ks --resolve passthrough.example.com:443:${EG_IP} \
  -o /dev/null -w "%{http_code}" https://passthrough.example.com/)

echo -e ""
echo -e "Results:"
echo -e "  Certificate Subject: ${EG_CERT_SUBJECT}"
echo -e "  HTTPS Status: ${EG_HTTPS}"
echo -e ""

if [ -n "$EG_CERT_SUBJECT" ] && [ "$EG_HTTPS" = "200" ]; then
  echo -e "✅ Envoy Gateway: SSL passthrough working!"
  echo -e ""
else
  echo -e "❌ Envoy Gateway: SSL passthrough test failed"
  echo -e ""
fi

echo -e ""
