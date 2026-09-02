#!/bin/sh
set -eu

# Regenerate the runtime config from env vars on every container start, so
# the same built image can point at different backends per environment.
# See public/config.js for the dev-mode equivalent.
CHAT_GATEWAY_HTTP_URL="${CHAT_GATEWAY_HTTP_URL:-http://localhost:8080}"
CHAT_GATEWAY_WS_URL="${CHAT_GATEWAY_WS_URL:-ws://localhost:8080}"
SUBSCRIPTION_SERVICE_URL="${SUBSCRIPTION_SERVICE_URL:-http://localhost:8081}"
NETWORK_OPS_SERVICE_URL="${NETWORK_OPS_SERVICE_URL:-http://localhost:8082}"

cat > /usr/share/nginx/html/config.js <<EOF
window.__CONFIG__ = {
  chatGatewayHttpUrl: "${CHAT_GATEWAY_HTTP_URL}",
  chatGatewayWsUrl: "${CHAT_GATEWAY_WS_URL}",
  subscriptionServiceUrl: "${SUBSCRIPTION_SERVICE_URL}",
  networkOpsServiceUrl: "${NETWORK_OPS_SERVICE_URL}",
};
EOF

exec "$@"
