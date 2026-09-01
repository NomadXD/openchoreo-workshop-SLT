# chat-agent

The LLM brain of the Vantage Mobile telco chat demo. A FastAPI service that wraps a
LangChain agent (`langchain.agents.create_agent`) over OpenAI, with a role-scoped tool
registry calling out to `subscription-service` and `network-ops-service`.

It trusts its caller completely: `chat-gateway` is expected to have already
authenticated the end user and resolved `role` / `actorId` / `targetCustomerId` before
calling this service. There is no auth, OAuth, or JWT verification here.

## Environment variables

| Variable                    | Required | Default        | Description                                              |
|------------------------------|:--------:|----------------|------------------------------------------------------------|
| `PORT`                       | no       | `8080`         | Port uvicorn listens on when run directly (not used by the Docker image, which always binds `8080`). |
| `OPENAI_API_KEY`             | **yes**  | —              | OpenAI API key. Startup fails fast with a clear error if unset. |
| `OPENAI_MODEL`               | no       | `gpt-4o-mini`  | OpenAI chat model id.                                       |
| `SUBSCRIPTION_SERVICE_URL`   | yes      | —              | Base URL of `subscription-service`, e.g. `http://subscription-service:8080`. |
| `NETWORK_OPS_SERVICE_URL`    | yes      | —              | Base URL of `network-ops-service`, e.g. `http://network-ops-service:8080`. |

`SUBSCRIPTION_SERVICE_URL` / `NETWORK_OPS_SERVICE_URL` are logged as warnings (not a
startup failure) if unset, since only the tools that call them would fail at request
time.

## Local run

With [`uv`](https://docs.astral.sh/uv/):

```sh
uv sync
OPENAI_API_KEY=sk-... \
SUBSCRIPTION_SERVICE_URL=http://localhost:8081 \
NETWORK_OPS_SERVICE_URL=http://localhost:8082 \
uv run uvicorn src.main:app --host 0.0.0.0 --port 8080
```

Without `uv` (plain venv):

```sh
python3 -m venv .venv && source .venv/bin/activate
pip install -e .
OPENAI_API_KEY=sk-... \
SUBSCRIPTION_SERVICE_URL=http://localhost:8081 \
NETWORK_OPS_SERVICE_URL=http://localhost:8082 \
python -m uvicorn src.main:app --host 0.0.0.0 --port 8080
```

Health check (no auth):

```sh
curl http://localhost:8080/healthz
# {"status":"ok"}
```

## The one real endpoint

`POST /agent/stream` takes the already-authenticated, already-scoped chat turn and
streams the assistant's reply as newline-delimited JSON
(`Content-Type: application/x-ndjson`), one compact object per line, flushed as it's
produced.

### Example — customer role

```sh
curl -N -X POST http://localhost:8080/agent/stream \
  -H "Content-Type: application/json" \
  -d '{
    "role": "customer",
    "actorId": "cust-001",
    "targetCustomerId": "cust-001",
    "conversationId": "conv-abc123",
    "message": "What plan am I on, and how much data have I used today?",
    "history": []
  }'
```

Streamed response (illustrative — exact token boundaries depend on the model):

```
{"type":"token","content":"Let"}
{"type":"token","content":" me"}
{"type":"token","content":" check"}
{"type":"token","content":" that"}
{"type":"token","content":" for"}
{"type":"token","content":" you."}
{"type":"tool_call","name":"get_my_subscription","args":{},"audit":false,"targetCustomerId":"cust-001"}
{"type":"tool_result","name":"get_my_subscription","result":{"planId":"plan-standard","planName":"Standard 20GB","status":"active"}}
{"type":"tool_call","name":"get_my_usage","args":{},"audit":false,"targetCustomerId":"cust-001"}
{"type":"tool_result","name":"get_my_usage","result":{"date":"2026-09-01","dataUsedGb":4.2,"dataLimitGb":20}}
{"type":"token","content":"You're"}
{"type":"token","content":" on"}
{"type":"token","content":" the"}
{"type":"token","content":" Standard"}
{"type":"token","content":" 20GB"}
{"type":"token","content":" plan"}
{"type":"token","content":" and"}
{"type":"token","content":" have"}
{"type":"token","content":" used"}
{"type":"token","content":" 4.2GB"}
{"type":"token","content":" of"}
{"type":"token","content":" your"}
{"type":"token","content":" 20GB"}
{"type":"token","content":" today."}
{"type":"done"}
```

### Example — employee role

```sh
curl -N -X POST http://localhost:8080/agent/stream \
  -H "Content-Type: application/json" \
  -d '{
    "role": "employee",
    "actorId": "agent-007",
    "conversationId": "conv-def456",
    "message": "Look up customer cust-002 and switch them to the premium plan.",
    "history": []
  }'
```

`targetCustomerId` is omitted here (the request isn't about any specific customer
until the employee names one); the tool calls carry their own `customer_id` argument
and the corresponding `tool_call` events report that as `targetCustomerId`.

## Role-scoped tools

A customer-role request only ever gets the customer tool set bound to the model —
the employee tools are never constructed for it, so the model has no catalog entry it
could even attempt to call:

- **Customer** (implicitly scoped to the caller's own `targetCustomerId`):
  `get_my_subscription`, `list_available_plans`, `change_my_subscription`,
  `get_my_usage`, `report_service_issue`.
- **Employee** (require an explicit `customer_id` argument): `lookup_customer`,
  `get_customer_account`, `change_customer_subscription`, `manage_plan_catalog`,
  `manage_service_reports`.

Every tool call forwards `X-Actor-Role` / `X-Actor-Id` headers to the backend
services for their own audit logging.

## Docker

```sh
docker build -t chat-agent:local .
docker run --rm -p 8080:8080 \
  -e OPENAI_API_KEY=sk-... \
  -e SUBSCRIPTION_SERVICE_URL=http://subscription-service:8080 \
  -e NETWORK_OPS_SERVICE_URL=http://network-ops-service:8080 \
  chat-agent:local
```
