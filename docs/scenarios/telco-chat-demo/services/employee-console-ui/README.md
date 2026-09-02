# employee-console-ui

A React + TypeScript SPA for the telco chat demo: an internal ops dashboard for support agents,
with three tabs.

- **Customers** — search/list customers (`subscription-service`), and view one customer's full
  account: profile, current plan, last 7 days' usage, and their service reports.
- **Incidents** — list/filter service reports (`network-ops-service`) by status/category, open one
  to see full detail, update its status/resolution notes, and see a **related incidents** panel
  (this customer's other reports, and other customers' reports in the same category).
- **Chat** — the assistant, over chat-gateway's WebSocket API, exactly as before.

Customers/Incidents call `subscription-service`/`network-ops-service` **directly** from the
browser — chat-gateway is chat-only, it isn't a proxy for these. Neither backend service requires
auth, so there's no token to attach; every dashboard request instead carries `X-Actor-Id: <agentId>`
so at least server logs are attributable to who made the call (see each service's README — this is
NOT a queryable audit trail, only the mutations that go through chat-gateway's chat path get one of
those). Clicking a report from the Customers tab jumps to it in Incidents; "Chat about this
customer" jumps to Chat with the "Assisting customer" field pre-filled.

There's no live-update mechanism — a manual "Refresh" button on each list is the whole mechanism,
by design (see the top-level demo README for why).

The "Assisting customer" field (Chat tab) must have a value before messages can be sent (the send
button is disabled with a hint until it does) — every outgoing message includes it as
`targetCustomerId`, since that's what scopes which customer's tools the assistant can act on.
Changing it mid-session drops a "— now assisting `<id>` —" divider into the transcript so it's
clear scope has changed rather than silently mixing customers.

## Develop

```sh
npm install
npm run dev
```

With no `window.__CONFIG__` present (i.e. `public/config.js` untouched),
the app talks to `http://localhost:8080` / `ws://localhost:8080` by default —
point a locally running chat-gateway at that port, or edit
`public/config.js` to point elsewhere.

## Build

```sh
npm run build   # tsc -b && vite build -> dist/
```

## Runtime configuration

The bundle is built once and reads its backend URL at container start, not
at build time, so the same image can be deployed against different
environments.

- `public/config.js` is a placeholder committed to the repo. It's loaded via
  `<script src="/config.js">` in `index.html`, before the app bundle, and sets
  `window.__CONFIG__ = { chatGatewayHttpUrl, chatGatewayWsUrl, subscriptionServiceUrl,
  networkOpsServiceUrl }`. The app reads those fields at startup (see `src/config.ts`), falling
  back to `localhost` defaults if `window.__CONFIG__` is missing entirely — this is what makes
  `npm run dev` work standalone.
- In the Docker image, `docker-entrypoint.sh` regenerates
  `/usr/share/nginx/html/config.js` from environment variables on **every**
  container start (not build time), then execs nginx:

  | Env var                     | Default                   |
  | ---------------------------- | -------------------------- |
  | `CHAT_GATEWAY_HTTP_URL`      | `http://localhost:8080`   |
  | `CHAT_GATEWAY_WS_URL`        | `ws://localhost:8080`     |
  | `SUBSCRIPTION_SERVICE_URL`   | `http://localhost:8081`   |
  | `NETWORK_OPS_SERVICE_URL`    | `http://localhost:8082`   |

## Docker

```sh
docker build -t employee-console-ui .
docker run -p 8080:8080 \
  -e CHAT_GATEWAY_HTTP_URL=http://chat-gateway.example.com \
  -e CHAT_GATEWAY_WS_URL=ws://chat-gateway.example.com \
  -e SUBSCRIPTION_SERVICE_URL=http://subscription-service.example.com \
  -e NETWORK_OPS_SERVICE_URL=http://network-ops-service.example.com \
  employee-console-ui
```

nginx listens on **8080** inside the container (see `nginx.conf`), not the
default 80.
