// Placeholder runtime config for local dev / `npm run dev`.
//
// In a built container, the entrypoint script overwrites this file at
// container start from CHAT_GATEWAY_HTTP_URL / CHAT_GATEWAY_WS_URL env vars.
// See Dockerfile + docker-entrypoint.sh.
window.__CONFIG__ = {
  chatGatewayHttpUrl: "http://localhost:8080",
  chatGatewayWsUrl: "ws://localhost:8080",
};
