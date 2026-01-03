#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
  echo -e ""
  echo -e "Cleaning up test resources..."

  # delete test resources
  kubectl delete -f ${SCRIPT_DIR}/../testdata/nginx/redirect.yaml 2>/dev/null || true
  kubectl delete -f ${SCRIPT_DIR}/../testdata/envoygateway/redirect.yaml 2>/dev/null || true
  kubectl delete secret ssl-redirect-tls -n ingress2eg 2>/dev/null || true

  echo -e "Cleanup completed."
}

trap cleanup EXIT

# Create TLS secret for ssl-redirect test
echo -e ""
echo -e "Creating TLS secret..."
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /tmp/tls.key -out /tmp/tls.crt \
  -subj "/CN=ssl-redirect.example.com" 2>/dev/null

kubectl create secret tls ssl-redirect-tls \
  -n ingress2eg \
  --key=/tmp/tls.key \
  --cert=/tmp/tls.crt \
  --dry-run=client -o yaml | kubectl apply -f - 2>/dev/null

# apply test resources
echo -e ""
echo -e "Applying test resources..."
kubectl apply -f ${SCRIPT_DIR}/../testdata/nginx/redirect.yaml
kubectl apply -f ${SCRIPT_DIR}/../testdata/envoygateway/redirect.yaml

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
  INGRESS_IP=$(kubectl get ingress -n ingress2eg redirect-ssl -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
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
echo -e "NGINX Ingress Redirect Tests"
echo -e "===================================="

# Test 1: SSL Redirect
echo -e ""
echo -e "Test 1: SSL Redirect (http -> https)"
RESOLVE_SSL="--resolve ssl-redirect.example.com:80:${INGRESS_IP}"
REDIRECT_SSL=$(curl -s ${RESOLVE_SSL} -w "\n%{redirect_url}|%{http_code}" http://ssl-redirect.example.com/ | tail -1)
REDIRECT_SSL_URL=$(echo "$REDIRECT_SSL" | cut -d'|' -f1)
REDIRECT_SSL_CODE=$(echo "$REDIRECT_SSL" | cut -d'|' -f2)

echo -e "  Redirect URL: ${REDIRECT_SSL_URL}"
echo -e "  Status Code: ${REDIRECT_SSL_CODE}"

if [[ "$REDIRECT_SSL_CODE" =~ ^(301|308)$ ]] && [ "$REDIRECT_SSL_URL" = "https://ssl-redirect.example.com/" ]; then
  echo -e "  ✅ SSL redirect working"
else
  echo -e "  ❌ SSL redirect failed"
fi

# Test 2: Force SSL Redirect
echo -e ""
echo -e "Test 2: Force SSL Redirect (http -> https, no TLS config)"
RESOLVE_FORCE="--resolve force-ssl-redirect.example.com:80:${INGRESS_IP}"
REDIRECT_FORCE=$(curl -s ${RESOLVE_FORCE} -w "\n%{redirect_url}|%{http_code}" http://force-ssl-redirect.example.com/ | tail -1)
REDIRECT_FORCE_URL=$(echo "$REDIRECT_FORCE" | cut -d'|' -f1)
REDIRECT_FORCE_CODE=$(echo "$REDIRECT_FORCE" | cut -d'|' -f2)

echo -e "  Redirect URL: ${REDIRECT_FORCE_URL}"
echo -e "  Status Code: ${REDIRECT_FORCE_CODE}"

if [[ "$REDIRECT_FORCE_CODE" =~ ^(301|308)$ ]] && [ "$REDIRECT_FORCE_URL" = "https://force-ssl-redirect.example.com/" ]; then
  echo -e "  ✅ Force SSL redirect working"
else
  echo -e "  ❌ Force SSL redirect failed"
fi

# Test 3: Permanent Redirect (301)
echo -e ""
echo -e "Test 3: Permanent Redirect (301 to different URL)"
RESOLVE_PERM="--resolve permanent-redirect.example.com:80:${INGRESS_IP}"
REDIRECT_PERM=$(curl -s ${RESOLVE_PERM} -w "\n%{redirect_url}|%{http_code}" http://permanent-redirect.example.com/ | tail -1)
REDIRECT_PERM_URL=$(echo "$REDIRECT_PERM" | cut -d'|' -f1)
REDIRECT_PERM_CODE=$(echo "$REDIRECT_PERM" | cut -d'|' -f2)

echo -e "  Redirect URL: ${REDIRECT_PERM_URL}"
echo -e "  Status Code: ${REDIRECT_PERM_CODE}"

if [ "$REDIRECT_PERM_CODE" = "301" ] && [ "$REDIRECT_PERM_URL" = "https://permanent.example.com/new-path" ]; then
  echo -e "  ✅ Permanent redirect working"
else
  echo -e "  ❌ Permanent redirect failed"
fi

# Test 4: Temporal Redirect (302)
echo -e ""
echo -e "Test 4: Temporal Redirect (302 to different URL)"
RESOLVE_TEMP="--resolve temporal-redirect.example.com:80:${INGRESS_IP}"
REDIRECT_TEMP=$(curl -s ${RESOLVE_TEMP} -w "\n%{redirect_url}|%{http_code}" http://temporal-redirect.example.com/ | tail -1)
REDIRECT_TEMP_URL=$(echo "$REDIRECT_TEMP" | cut -d'|' -f1)
REDIRECT_TEMP_CODE=$(echo "$REDIRECT_TEMP" | cut -d'|' -f2)

echo -e "  Redirect URL: ${REDIRECT_TEMP_URL}"
echo -e "  Status Code: ${REDIRECT_TEMP_CODE}"

if [ "$REDIRECT_TEMP_CODE" = "302" ] && [ "$REDIRECT_TEMP_URL" = "https://temporal.example.com/temp-path" ]; then
  echo -e "  ✅ Temporal redirect working"
else
  echo -e "  ❌ Temporal redirect failed"
fi

echo -e ""
echo -e "===================================="
echo -e "Envoy Gateway Redirect Tests"
echo -e "===================================="

# Test 1: SSL Redirect
echo -e ""
echo -e "Test 1: SSL Redirect (http -> https)"
RESOLVE_SSL="--resolve ssl-redirect.example.com:80:${EG_IP}"
REDIRECT_SSL=$(curl -s ${RESOLVE_SSL} -w "\n%{redirect_url}|%{http_code}" http://ssl-redirect.example.com/ | tail -1)
REDIRECT_SSL_URL=$(echo "$REDIRECT_SSL" | cut -d'|' -f1)
REDIRECT_SSL_CODE=$(echo "$REDIRECT_SSL" | cut -d'|' -f2)

echo -e "  Redirect URL: ${REDIRECT_SSL_URL}"
echo -e "  Status Code: ${REDIRECT_SSL_CODE}"

if [ "$REDIRECT_SSL_CODE" = "301" ] && [ "$REDIRECT_SSL_URL" = "https://ssl-redirect.example.com/" ]; then
  echo -e "  ✅ SSL redirect working"
else
  echo -e "  ❌ SSL redirect failed"
fi

# Test 2: Force SSL Redirect
echo -e ""
echo -e "Test 2: Force SSL Redirect (http -> https, no TLS config)"
RESOLVE_FORCE="--resolve force-ssl-redirect.example.com:80:${EG_IP}"
REDIRECT_FORCE=$(curl -s ${RESOLVE_FORCE} -w "\n%{redirect_url}|%{http_code}" http://force-ssl-redirect.example.com/ | tail -1)
REDIRECT_FORCE_URL=$(echo "$REDIRECT_FORCE" | cut -d'|' -f1)
REDIRECT_FORCE_CODE=$(echo "$REDIRECT_FORCE" | cut -d'|' -f2)

echo -e "  Redirect URL: ${REDIRECT_FORCE_URL}"
echo -e "  Status Code: ${REDIRECT_FORCE_CODE}"

if [ "$REDIRECT_FORCE_CODE" = "301" ] && [ "$REDIRECT_FORCE_URL" = "https://force-ssl-redirect.example.com/" ]; then
  echo -e "  ✅ Force SSL redirect working"
else
  echo -e "  ❌ Force SSL redirect failed"
fi

# Test 3: Permanent Redirect (301)
echo -e ""
echo -e "Test 3: Permanent Redirect (301 to different URL)"
RESOLVE_PERM="--resolve permanent-redirect.example.com:80:${EG_IP}"
REDIRECT_PERM=$(curl -s ${RESOLVE_PERM} -w "\n%{redirect_url}|%{http_code}" http://permanent-redirect.example.com/ | tail -1)
REDIRECT_PERM_URL=$(echo "$REDIRECT_PERM" | cut -d'|' -f1)
REDIRECT_PERM_CODE=$(echo "$REDIRECT_PERM" | cut -d'|' -f2)

echo -e "  Redirect URL: ${REDIRECT_PERM_URL}"
echo -e "  Status Code: ${REDIRECT_PERM_CODE}"

if [ "$REDIRECT_PERM_CODE" = "301" ] && [ "$REDIRECT_PERM_URL" = "https://permanent.example.com/new-path" ]; then
  echo -e "  ✅ Permanent redirect working"
else
  echo -e "  ❌ Permanent redirect failed"
fi

# Test 4: Temporal Redirect (302)
echo -e ""
echo -e "Test 4: Temporal Redirect (302 to different URL)"
RESOLVE_TEMP="--resolve temporal-redirect.example.com:80:${EG_IP}"
REDIRECT_TEMP=$(curl -s ${RESOLVE_TEMP} -w "\n%{redirect_url}|%{http_code}" http://temporal-redirect.example.com/ | tail -1)
REDIRECT_TEMP_URL=$(echo "$REDIRECT_TEMP" | cut -d'|' -f1)
REDIRECT_TEMP_CODE=$(echo "$REDIRECT_TEMP" | cut -d'|' -f2)

echo -e "  Redirect URL: ${REDIRECT_TEMP_URL}"
echo -e "  Status Code: ${REDIRECT_TEMP_CODE}"

if [ "$REDIRECT_TEMP_CODE" = "302" ] && [ "$REDIRECT_TEMP_URL" = "https://temporal.example.com/temp-path" ]; then
  echo -e "  ✅ Temporal redirect working"
else
  echo -e "  ❌ Temporal redirect failed"
fi

echo -e ""
