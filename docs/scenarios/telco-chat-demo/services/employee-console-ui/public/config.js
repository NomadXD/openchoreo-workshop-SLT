// Placeholder runtime config for local dev / `npm run dev`.
//
// In a built container, the entrypoint script overwrites this file at
// container start from env vars. See Dockerfile + docker-entrypoint.sh.
window.__CONFIG__ = {
  chatGatewayHttpUrl: "http://localhost:8080",
  chatGatewayWsUrl: "ws://localhost:8080",
  subscriptionServiceUrl: "http://localhost:8081",
  networkOpsServiceUrl: "http://localhost:8082",
};
