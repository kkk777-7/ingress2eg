#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/proxy-ssl.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/proxy-ssl.yaml 2>/dev/null || true
  kubectl delete secret backend-client-cert -n ingress2eg 2>/dev/null || true

  # delete temporary certificate files
  rm -f /tmp/backend-ca.crt /tmp/backend-ca.key
  rm -f /tmp/backend-client.key /tmp/backend-client.csr /tmp/backend-client.crt

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# Generate client certificate for mTLS
echo -e ""
echo -e "Generating client certificate for mTLS..."

if ! kubectl get secret nginx-tls-ca-secret -n ingress2eg &>/dev/null; then
  echo -e "❌ Error: nginx-tls-ca-secret not found"
  exit 1
fi

# Extract CA certificate and key from nginx-tls-ca-secret
kubectl get secret nginx-tls-ca-secret -n ingress2eg -o jsonpath='{.data.tls\.crt}' | base64 -d > /tmp/backend-ca.crt
kubectl get secret nginx-tls-ca-secret -n ingress2eg -o jsonpath='{.data.tls\.key}' | base64 -d > /tmp/backend-ca.key

# Generate client certificate request
openssl req -nodes -newkey rsa:2048 \
  -keyout /tmp/backend-client.key -out /tmp/backend-client.csr \
  -subj "/CN=nginx-ingress-client" 2>/dev/null

# Sign client certificate with the CA
openssl x509 -req -days 365 \
  -in /tmp/backend-client.csr \
  -CA /tmp/backend-ca.crt -CAkey /tmp/backend-ca.key -CAcreateserial \
  -out /tmp/backend-client.crt 2>/dev/null

# Create secret with client cert and CA cert
kubectl create secret generic backend-client-cert \
  -n ingress2eg \
  --from-file=tls.crt=/tmp/backend-client.crt \
  --from-file=tls.key=/tmp/backend-client.key \
  --from-file=ca.crt=/tmp/backend-ca.crt \
  --dry-run=client -o yaml | kubectl apply -f - 2>/dev/null

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/proxy-ssl.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/proxy-ssl.yaml

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
  INGRESS_IP=$(kubectl get ingress -n ingress2eg proxy-ssl -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
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
echo -e "NGINX Ingress Proxy SSL Test"
echo -e "===================================="

RESOLVE="--resolve proxy-ssl.example.com:80:${INGRESS_IP}"

echo -e ""
echo -e "Testing proxy SSL connection to backend..."
RESPONSE=$(curl -s ${RESOLVE} -w "\n%{http_code}" http://proxy-ssl.example.com/ | tail -1)

echo -e "  Status Code: ${RESPONSE}"

if [ "$RESPONSE" = "200" ]; then
  echo -e "  ✅ Proxy SSL connection working"
else
  echo -e "  ❌ Proxy SSL connection failed (expected 200, got ${RESPONSE})"
fi

echo -e ""
echo -e "===================================="
echo -e "Envoy Gateway Proxy SSL Test"
echo -e "===================================="

RESOLVE="--resolve proxy-ssl.example.com:80:${EG_IP}"

echo -e ""
echo -e "Testing proxy SSL connection to backend..."
RESPONSE=$(curl -s ${RESOLVE} -w "\n%{http_code}" http://proxy-ssl.example.com/ | tail -1)

echo -e "  Status Code: ${RESPONSE}"

if [ "$RESPONSE" = "200" ]; then
  echo -e "  ✅ Proxy SSL connection working"
else
  echo -e "  ❌ Proxy SSL connection failed (expected 200, got ${RESPONSE})"
fi

echo -e ""
