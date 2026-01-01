#!/usr/bin/env bash

set -euo pipefail

# Envoy Gateway version
EG_VERSION="v1.6.1"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EG_VALUES_FILE="${SCRIPT_DIR}/eg-values.yaml"

helm template eg oci://docker.io/envoyproxy/gateway-crds-helm \
  --version "${EG_VERSION}" \
  --set crds.gatewayAPI.enabled=true \
  --set crds.gatewayAPI.channel=standard \
  --set crds.envoyGateway.enabled=true \
  | kubectl apply --server-side -f -

helm install eg oci://docker.io/envoyproxy/gateway-helm \
  --version "${EG_VERSION}" \
  -n envoy-gateway-system \
  --create-namespace \
  -f "${EG_VALUES_FILE}" \
  --skip-crds

kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: eg
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
EOF