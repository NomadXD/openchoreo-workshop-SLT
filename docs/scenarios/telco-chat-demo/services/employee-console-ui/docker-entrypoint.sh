#!/bin/sh
set -eu

# Regenerate the runtime config from env vars on every container start, so
# the same built image can point at a different chat-gateway per
# environment. See public/config.js for the dev-mode equivalent.
CHAT_GATEWAY_HTTP_URL="${CHAT_GATEWAY_HTTP_URL:-http://localhost:8080}"
CHAT_GATEWAY_WS_URL="${CHAT_GATEWAY_WS_URL:-ws://localhost:8080}"

cat > /usr/share/nginx/html/config.js <<EOF
window.__CONFIG__ = {
  chatGatewayHttpUrl: "${CHAT_GATEWAY_HTTP_URL}",
  chatGatewayWsUrl: "${CHAT_GATEWAY_WS_URL}",
};
EOF

exec "$@"
