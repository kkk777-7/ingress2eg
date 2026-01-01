#!/usr/bin/env bash

set -euo pipefail

INGRESS_IP=$(kubectl get ingress -n ingress2eg basic-auth -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
EG_IP