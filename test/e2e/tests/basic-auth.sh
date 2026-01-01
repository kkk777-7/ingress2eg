#!/usr/bin/env bash

set -euo pipefail

# apply test resources
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/basic-auth.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/basic-auth.yaml

echo -e ""
echo -e "Applying test resources and waiting for IPs..."
echo -e ""

# check for INGRESS_IP and EG_IP
INGRESS_IP=$(kubectl get ingress -n ingress2eg basic-auth -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
EG_IP=$(kubectl get gateway -n ingress2eg nginx -o jsonpath='{.status.addresses[0].value}' 2>/dev/null)

# wait for INGRESS_IP with timeout
TIMEOUT=300  # 5 minutes
ELAPSED=0
while [ -z "$INGRESS_IP" ] && [ $ELAPSED -lt $TIMEOUT ]; do
  echo -e "Waiting for INGRESS_IP... (${ELAPSED}s/${TIMEOUT}s)"
  sleep 2
  ELAPSED=$((ELAPSED + 2))
  INGRESS_IP=$(kubectl get ingress -n ingress2eg basic-auth -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
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
echo -e "NGINX Ingress Basic Auth Test"
echo -e "===================================="

INGRESS_RESOLVE="--resolve foo.bar.com:443:${INGRESS_IP} --resolve foo.bar.com:80:${INGRESS_IP}"

INGRESS_BASIC_NO_AUTH=$(curl -s ${INGRESS_RESOLVE} -o /dev/null -w "%{http_code}" http://foo.bar.com/basic)
INGRESS_BASIC_VALID=$(curl -s ${INGRESS_RESOLVE} -o /dev/null -w "%{http_code}" -u foo:bar http://foo.bar.com/basic)
INGRESS_BASIC_INVALID=$(curl -s ${INGRESS_RESOLVE} -o /dev/null -w "%{http_code}" -u foo:wrong http://foo.bar.com/basic)

echo -e ""
echo -e "Results:"
echo -e "  No Auth: ${INGRESS_BASIC_NO_AUTH}"
echo -e "  Valid Credentials: ${INGRESS_BASIC_VALID}"
echo -e "  Invalid Credentials: ${INGRESS_BASIC_INVALID}"
echo -e ""

if [ "$INGRESS_BASIC_NO_AUTH" = "401" ] && [ "$INGRESS_BASIC_VALID" = "200" ] && [ "$INGRESS_BASIC_INVALID" = "401" ]; then
  echo -e "✅ NGINX Ingress: Basic auth working!"
  echo -e ""
else
  echo -e "❌ NGINX Ingress: Basic auth failed"
  echo -e ""
fi

echo -e "===================================="
echo -e "Envoy Gateway Basic Auth Test"
echo -e "===================================="

EG_RESOLVE="--resolve foo.bar.com:443:${EG_IP} --resolve foo.bar.com:80:${EG_IP}"

EG_BASIC_NO_AUTH=$(curl -s ${EG_RESOLVE} -o /dev/null -w "%{http_code}" http://foo.bar.com/basic)
EG_BASIC_VALID=$(curl -s ${EG_RESOLVE} -o /dev/null -w "%{http_code}" -u foo:bar http://foo.bar.com/basic)
EG_BASIC_INVALID=$(curl -s ${EG_RESOLVE} -o /dev/null -w "%{http_code}" -u foo:wrong http://foo.bar.com/basic)

echo -e ""
echo -e "Results:"
echo -e "  No Auth: ${EG_BASIC_NO_AUTH}"
echo -e "  Valid Credentials: ${EG_BASIC_VALID}"
echo -e "  Invalid Credentials: ${EG_BASIC_INVALID}"
echo -e ""

if [ "$EG_BASIC_NO_AUTH" = "401" ] && [ "$EG_BASIC_VALID" = "200" ] && [ "$EG_BASIC_INVALID" = "401" ]; then
  echo -e "✅ Envoy Gateway: Basic auth working!"
  echo -e ""
else
  echo -e "❌ Envoy Gateway: Basic auth failed"
  echo -e ""
fi

# cleanup test resources
kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/basic-auth.yaml
kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/basic-auth.yaml

echo -e ""
echo -e "Cleanup test resources completed."