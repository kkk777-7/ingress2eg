#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_DIR=$(mktemp -d)

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/auth-tls-secret.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/auth-tls-secret.yaml 2>/dev/null || true

  # delete secrets
  kubectl delete secret -n ingress2eg ca-secret 2>/dev/null || true
  kubectl delete secret -n ingress2eg tls-secret 2>/dev/null || true

  # remove temp cert directory
  rm -rf "$CERT_DIR"

  echo -e "Cleanup completed."
}

trap cleanup EXIT

echo -e ""
echo -e "===================================="
echo -e "Generating Test Certificates"
echo -e "===================================="
echo -e ""

# generate CA key and certificate
openssl req -x509 -newkey rsa:4096 -keyout ${CERT_DIR}/ca-key.pem -out ${CERT_DIR}/ca-cert.pem \
  -days 365 -nodes -subj "/CN=Test CA" 2>/dev/null

# generate server key and certificate
openssl req -newkey rsa:4096 -keyout ${CERT_DIR}/server-key.pem -out ${CERT_DIR}/server-req.pem \
  -nodes -subj "/CN=mtls.example.com" 2>/dev/null

openssl x509 -req -in ${CERT_DIR}/server-req.pem -CA ${CERT_DIR}/ca-cert.pem \
  -CAkey ${CERT_DIR}/ca-key.pem -CAcreateserial -out ${CERT_DIR}/server-cert.pem -days 365 2>/dev/null

# generate valid client key and certificate
openssl req -newkey rsa:4096 -keyout ${CERT_DIR}/client-key.pem -out ${CERT_DIR}/client-req.pem \
  -nodes -subj "/CN=valid-client" 2>/dev/null

openssl x509 -req -in ${CERT_DIR}/client-req.pem -CA ${CERT_DIR}/ca-cert.pem \
  -CAkey ${CERT_DIR}/ca-key.pem -CAcreateserial -out ${CERT_DIR}/client-cert.pem -days 365 2>/dev/null

# generate invalid client key and certificate (self-signed, not from CA)
openssl req -x509 -newkey rsa:4096 -keyout ${CERT_DIR}/invalid-client-key.pem \
  -out ${CERT_DIR}/invalid-client-cert.pem -days 365 -nodes -subj "/CN=invalid-client" 2>/dev/null

echo -e "✅ Certificates generated in ${CERT_DIR}"

# create secrets
echo -e ""
echo -e "Creating Kubernetes secrets..."

kubectl create secret generic ca-secret -n ingress2eg \
  --from-file=ca.crt=${CERT_DIR}/ca-cert.pem \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret tls tls-secret -n ingress2eg \
  --cert=${CERT_DIR}/server-cert.pem \
  --key=${CERT_DIR}/server-key.pem \
  --dry-run=client -o yaml | kubectl apply -f -

echo -e "✅ Secrets created"

# apply test resources
echo -e ""
echo -e "Applying test resources and waiting for IPs..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/auth-tls-secret.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/auth-tls-secret.yaml

# check for INGRESS_IP and EG_IP
INGRESS_IP=$(kubectl get ingress -n ingress2eg auth-tls -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
EG_IP=$(kubectl get gateway -n ingress2eg nginx -o jsonpath='{.status.addresses[0].value}' 2>/dev/null)

# wait for INGRESS_IP with timeout
TIMEOUT=300  # 5 minutes
ELAPSED=0
while [ -z "$INGRESS_IP" ] && [ $ELAPSED -lt $TIMEOUT ]; do
  echo -e "Waiting for INGRESS_IP... (${ELAPSED}s/${TIMEOUT}s)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  INGRESS_IP=$(kubectl get ingress -n ingress2eg auth-tls -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
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
echo -e "NGINX Ingress mTLS Test"
echo -e "===================================="

INGRESS_RESOLVE="--resolve mtls.example.com:443:${INGRESS_IP} --resolve mtls.example.com:80:${INGRESS_IP}"

# test without client certificate (should fail)
INGRESS_NO_CERT=$(curl -ks ${INGRESS_RESOLVE} -o /dev/null -w "%{http_code}" https://mtls.example.com/secure)

# test with valid client certificate (should succeed)
INGRESS_VALID_CERT=$(curl -ks ${INGRESS_RESOLVE} \
  --cert ${CERT_DIR}/client-cert.pem \
  --key ${CERT_DIR}/client-key.pem \
  -o /dev/null -w "%{http_code}" https://mtls.example.com/secure)

# test with invalid client certificate (should fail)
INGRESS_INVALID_CERT=$(curl -ks ${INGRESS_RESOLVE} \
  --cert ${CERT_DIR}/invalid-client-cert.pem \
  --key ${CERT_DIR}/invalid-client-key.pem \
  -o /dev/null -w "%{http_code}" https://mtls.example.com/secure)

echo -e ""
echo -e "Results:"
echo -e "  No Client Cert: ${INGRESS_NO_CERT}"
echo -e "  Valid Client Cert: ${INGRESS_VALID_CERT}"
echo -e "  Invalid Client Cert: ${INGRESS_INVALID_CERT}"
echo -e ""

if [ "$INGRESS_NO_CERT" = "400" ] && [ "$INGRESS_VALID_CERT" = "200" ] && [ "$INGRESS_INVALID_CERT" = "400" ]; then
  echo -e "✅ NGINX Ingress: mTLS working correctly!"
  echo -e ""
else
  echo -e "❌ NGINX Ingress: mTLS failed"
  echo -e ""
fi

echo -e "===================================="
echo -e "Envoy Gateway mTLS Test"
echo -e "===================================="

EG_RESOLVE="--resolve mtls.example.com:443:${EG_IP} --resolve mtls.example.com:80:${EG_IP}"

# test without client certificate (should fail)
EG_NO_CERT=$(curl -ks ${EG_RESOLVE} -o /dev/null -w "%{http_code}" https://mtls.example.com/secure || true)

# test with valid client certificate (should succeed)
EG_VALID_CERT=$(curl -ks ${EG_RESOLVE} \
  --cert ${CERT_DIR}/client-cert.pem \
  --key ${CERT_DIR}/client-key.pem \
  -o /dev/null -w "%{http_code}" https://mtls.example.com/secure)

# test with invalid client certificate (should fail)
EG_INVALID_CERT=$(curl -ks ${EG_RESOLVE} \
  --cert ${CERT_DIR}/invalid-client-cert.pem \
  --key ${CERT_DIR}/invalid-client-key.pem \
  -o /dev/null -w "%{http_code}" https://mtls.example.com/secure || true)

echo -e ""
echo -e "Results:"
echo -e "  No Client Cert: ${EG_NO_CERT}"
echo -e "  Valid Client Cert: ${EG_VALID_CERT}"
echo -e "  Invalid Client Cert: ${EG_INVALID_CERT}"
echo -e ""

# Envoy Gateway rejects invalid certs at TLS handshake level
# This causes connection failure, resulting in curl returning 000
if [ "$EG_NO_CERT" = "000" ] && [ "$EG_VALID_CERT" = "200" ] && [ "$EG_INVALID_CERT" = "000" ]; then
  echo -e "✅ Envoy Gateway: mTLS working correctly!"
  echo -e ""
else
  echo -e "❌ Envoy Gateway: mTLS test failed"
  echo -e ""
fi
