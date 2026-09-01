# chat-gateway

Browser-facing BFF for the telco chat demo. It handles mock login (issues JWTs), owns the
WebSocket chat connection, persists conversations in Postgres, forwards each chat turn to the
internal `chat-agent` service over HTTP, streams the response back to the browser, coordinates
via Redis pub/sub, and writes an audit log for audited tool calls.

## Environment variables

| Var              | Default                    | Notes                                                        |
|-------------------|----------------------------|---------------------------------------------------------------|
| `PORT`            | `8080`                     | HTTP/WS listen port                                            |
| `DATABASE_URL`    | *(required)*               | Postgres DSN for chat-gateway's own `chat-db`                  |
| `REDIS_URL`       | *(required)*               | Full Redis connection URL, e.g. `redis://:<password>@host:6379` — parsed with `redis.ParseURL`, so a password-protected instance (the `valkey` `Resource`) works the same as a bare no-auth one (`redis://host:6379`) |
| `CHAT_AGENT_URL`  | *(required)*               | Base URL of the chat-agent service, e.g. `http://chat-agent:8080` |
| `JWT_SECRET`      | `dev-secret-change-me`     | HMAC signing secret. A warning is logged if left unset — **do not rely on the default outside this demo.** |

## Local run

```sh
# Postgres + Redis, e.g. via docker:
docker run -d --name chat-db -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=chatdb -p 5432:5432 postgres:16-alpine
docker run -d --name chat-redis -p 6379:6379 redis:7-alpine

export DATABASE_URL="postgres://postgres:postgres@localhost:5432/chatdb?sslmode=disable"
export REDIS_URL="redis://localhost:6379"   # no password against the bare redis:7-alpine image above
export CHAT_AGENT_URL="http://localhost:9090"   # point at a running chat-agent
export JWT_SECRET="dev-secret-change-me"

go run .
```

The Postgres schema (`conversations`, `messages`, `audit_log`) is created automatically on
startup via idempotent `CREATE TABLE IF NOT EXISTS` statements — no separate migration step.

### Docker

```sh
docker build -t chat-gateway .
docker run --rm -p 8080:8080 \
  -e DATABASE_URL="postgres://postgres:postgres@host.docker.internal:5432/chatdb?sslmode=disable" \
  -e REDIS_URL="redis://host.docker.internal:6379" \
  -e CHAT_AGENT_URL="http://chat-agent:8080" \
  -e JWT_SECRET="dev-secret-change-me" \
  chat-gateway
```

## API

### Health

```sh
curl http://localhost:8080/healthz
```

### Mock login

No password — this is a demo. Any non-empty id is accepted; `chat-gateway` does not itself
validate customer/agent ids against any external directory (subscription-service is the source
of truth for valid customers, and this gateway is deliberately kept unaware of it).

```sh
curl -X POST http://localhost:8080/api/auth/customer/login \
  -d '{"customerId":"cust-001"}'
# => {"token":"<jwt>"}

curl -X POST http://localhost:8080/api/auth/employee/login \
  -d '{"agentId":"agent-007"}'
# => {"token":"<jwt>"}
```

Tokens are HS256, signed with `JWT_SECRET`, carry `sub` (customerId/agentId), `role`
(`customer`/`employee`), and expire after 24h.

### WebSocket chat

Connect with the token as a query parameter. The upgrade is rejected with `401` if the token is
missing, malformed, or expired.

```sh
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/customer/login -d '{"customerId":"cust-001"}' | jq -r .token)
wscat -c "ws://localhost:8080/ws/chat?token=$TOKEN"
```

Send a chat turn as a JSON text frame:

```json
{"type":"message","content":"What's my current bill?"}
```

- Omit `conversationId` to start a new conversation (its id, `conv-<uuid>`, then sticks for the
  rest of the connection unless a later message explicitly supplies a different one).
- Customers: `targetCustomerId` is always forced server-side to the JWT's `sub`, regardless of
  what the client sends.
- Employees: `targetCustomerId` is required and non-empty, or the gateway replies
  `{"type":"error","message":"targetCustomerId is required for employee turns"}` without
  forwarding anything.

The gateway streams back whatever `chat-agent` emits, forwarded verbatim, one WS text frame per
NDJSON line: `token`, `tool_call`, `tool_result`, `done`, or `error`. `token` events are also
published on the Redis pub/sub channel `conv:{conversationId}`. `tool_call` events with
`"audit":true` are recorded to `audit_log`. On `done`, the assembled assistant reply (the
concatenation of all `token.content` since the last `done`) is persisted as a `messages` row.

Rate limiting: 20 turns per 60s window per actor (`ratelimit:{actorId}` in Redis). Exceeding it
returns `{"type":"error","message":"rate limit exceeded, try again shortly"}` without forwarding
to chat-agent (the connection stays open).

If `chat-agent` itself is unreachable or returns a non-2xx status before streaming starts, the
gateway replies `{"type":"error","message":"agent unavailable"}` and keeps the connection open.

### Conversation history

```sh
curl http://localhost:8080/api/conversations/conv-<uuid>/messages
# => [{"role":"user","content":"...","createdAt":"..."}, ...]
```

**Known simplification:** this endpoint performs no authorization check — any caller can read any
conversation's messages by id. That's acceptable for this demo; a real deployment would need to
verify the caller is the conversation's subject or an authorized employee before returning data.

## Design notes / deviations from the base contract

- **`conversations.subject_role` / `subject_id`** are not specified precisely by the contract
  beyond the column names. This implementation always sets `subject_role="customer"` and
  `subject_id=<the resolved targetCustomerId>` — i.e. a conversation's "subject" is always the
  customer it concerns, whether the customer is chatting directly or an employee is assisting
  them. This keeps conversations addressable/filterable by customer regardless of who's typing.
- **Redis pub/sub payload**: the contract says to "publish it" for `token` events without
  specifying the exact payload shape. This implementation publishes the full raw NDJSON line
  (the same bytes forwarded to the browser) rather than just the bare content string, so a
  subscriber gets the complete event (type + content) with no extra parsing assumptions.
- **`audit_log.details`**: populated directly from the `tool_call` event's `args` field (falls
  back to `{}` if absent), matching "the tool name and args as a JSON details blob" in the spec.
- **Unknown/malformed client WS frames**: any frame whose `type` isn't `"message"`, or that
  fails to parse as JSON, gets a generic `{"type":"error","message":"..."}` reply rather than
  closing the connection — the contract only specifies the `"message"` frame type, so this is an
  additive robustness behavior, not a deviation from any specified path.
- No connection-level ping/pong keepalive or per-message write deadlines are implemented; not
  required by the contract and kept out to minimize demo surface area.
