#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

#
# Runs Envoy and the example control plane server.  See
# `internal/example` for the go source.
#

#Envoy start-up command
ENVOY=${ENVOY:-/usr/local/bin/envoy}

# Start envoy: important to keep drain time short
(envoy -c config/ingress-gw.yaml --base-id 1 --drain-time-s 1 -l trace --component-log-level upstream:info)&
ENVOY_PID_INGRESS=$!
(envoy -c config/worker-sidecar.yaml --base-id 2 --drain-time-s 1 -l info)&
ENVOY_PID_WORKER=$!


function cleanup() {
  kill ${ENVOY_PID_INGRESS}
  kill ${ENVOY_PID_WORKER}
}
trap cleanup EXIT

#Run the control plane
bin/controlplane $@
