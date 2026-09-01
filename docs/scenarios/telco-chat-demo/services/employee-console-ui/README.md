# employee-console-ui

A small React + TypeScript SPA for the telco chat demo. Lets a support agent
sign in with an agent id, pick which customer they're currently assisting,
and chat with the assistant over the chat-gateway's WebSocket API, with tool
calls surfaced inline.

The "Assisting customer" field must have a value before messages can be
sent (the send button is disabled with a hint until it does) — every
outgoing message includes it as `targetCustomerId`, since that's what scopes
which customer's tools the assistant can act on. Changing it mid-session
drops a "— now assisting `<id>` —" divider into the transcript so it's clear
scope has changed rather than silently mixing customers.

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
  `<script src="/config.js">` in `index.html`, before the app bundle, and
  sets `window.__CONFIG__ = { chatGatewayHttpUrl, chatGatewayWsUrl }`. The
  app reads those two fields at startup (see `src/config.ts`), falling back
  to `http://localhost:8080` / `ws://localhost:8080` if `window.__CONFIG__`
  is missing entirely — this is what makes `npm run dev` work standalone.
- In the Docker image, `docker-entrypoint.sh` regenerates
  `/usr/share/nginx/html/config.js` from environment variables on **every**
  container start (not build time), then execs nginx:

  | Env var                 | Default                  |
  | ------------------------ | ------------------------- |
  | `CHAT_GATEWAY_HTTP_URL`  | `http://localhost:8080`  |
  | `CHAT_GATEWAY_WS_URL`    | `ws://localhost:8080`    |

## Docker

```sh
docker build -t employee-console-ui .
docker run -p 8080:8080 \
  -e CHAT_GATEWAY_HTTP_URL=http://chat-gateway.example.com \
  -e CHAT_GATEWAY_WS_URL=ws://chat-gateway.example.com \
  employee-console-ui
```

nginx listens on **8080** inside the container (see `nginx.conf`), not the
default 80.
