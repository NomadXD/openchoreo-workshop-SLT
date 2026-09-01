# Telco Chat Demo — Vantage Mobile

A role-aware support chat agent for a fictitious mobile carrier, **Vantage Mobile**, spread across
three OpenChoreo projects. Customers self-serve (check their plan, upgrade/downgrade, check usage,
report an outage); employees get the same chat as an "agent console" with cross-account tools —
looking up any customer, managing their subscription on their behalf, and managing the plan
catalog itself. One tool router (`chat-agent`), two very different grants, gated by the caller's
role.

> **Prerequisite:** a running OpenChoreo environment (all four planes) — see the
> [installation guide](../../installation/README.md) — plus the
> **[workflow plane](../../installation/05-workflow-plane.md)** is *not* required here: every
> component in this demo is deployed from a pre-built local image, not built from source.

## Layout

```
telco-chat-demo/
├── portal/                     # Project: portal — frontends only
│   ├── project.yaml
│   └── components/
│       ├── customer-portal-ui.yaml
│       └── employee-console-ui.yaml
├── support-agent/               # Project: support-agent — the chat layer
│   ├── project.yaml
│   ├── resources/
│   │   ├── chat-db.yaml
│   │   └── chat-cache.yaml       # Valkey (Redis-protocol) Resource
│   └── components/
│       ├── chat-gateway.yaml
│       └── chat-agent.yaml
├── telco-services/               # Project: telco-services — system of record
│   ├── project.yaml
│   ├── resources/telco-db.yaml
│   └── components/
│       ├── subscription-service.yaml    # BSS: accounts, plans, subscriptions
│       └── network-ops-service.yaml     # OSS: usage, service-disruption reports
└── services/                     # All source code, one directory per component
    ├── customer-portal-ui/
    ├── employee-console-ui/
    ├── chat-gateway/
    ├── chat-agent/
    ├── subscription-service/
    └── network-ops-service/
```

Each project owns a disjoint slice: Portal has no backend logic at all; Support Agent owns the
conversational layer and its own storage (`chat-db`, `chat-cache`); Telco Services owns the
customer/subscription/usage/incident data both `chat-agent`'s tools and (in a real system) other
back-office systems would read from. `chat-agent` is the one component that reaches across a
project boundary — see **Verify the cross-project hop** below before you rely on it.

`chat-cache` is a `Resource` bound to the `valkey` `ClusterResourceType` — confirmed available on
this cluster alongside `postgres` and `nats`. Its Valkey instance always runs with a generated
password (there's no config knob to disable auth), so `chat-gateway` binds the resource's `url`
output (a full `redis://:<password>@host:6379` connection string) to `REDIS_URL` and parses it with
`redis.ParseURL` — see `services/chat-gateway/config.go` / `main.go`.

## 1 — Build the images and load them into the cluster

Each service under `services/` has its own `README.md` with exact env vars and local-run
instructions. Build and import all six:

```bash
cd services
for svc in subscription-service network-ops-service chat-gateway chat-agent \
           customer-portal-ui employee-console-ui; do
  docker build -t "telco-chat-demo/$svc:dev" "./$svc"
  k3d image import "telco-chat-demo/$svc:dev" -c openchoreo
done
```

(Adjust the `-c openchoreo` cluster name if your install used a different one — see
[Step 1 of the installation guide](../../installation/01-create-cluster.md).)

## 2 — Deploy Telco Services first (nothing else depends on it upstream)

```bash
kubectl apply -f telco-services/project.yaml
kubectl apply -f telco-services/resources/telco-db.yaml
```

Wait for the `telco-db` Resource to produce a `ResourceRelease`, then promote the binding to it
(see the comment in `telco-db.yaml`):

```bash
occ resource promote --env development telco-db
```

Then the two services:

```bash
kubectl apply -f telco-services/components/subscription-service.yaml
kubectl apply -f telco-services/components/network-ops-service.yaml
```

## 3 — Deploy Support Agent

```bash
kubectl apply -f support-agent/project.yaml
kubectl apply -f support-agent/resources/chat-db.yaml
occ resource promote --env development chat-db
kubectl apply -f support-agent/resources/chat-cache.yaml
occ resource promote --env development chat-cache
```

**Provide the OpenAI key** before deploying `chat-agent` — its Workload reads it from a Secret this
manifest set deliberately doesn't create for you (don't check an API key into git). Find
`chat-agent`'s rendered data-plane namespace and create the secret there:

```bash
kubectl get ns | grep support-agent-development     # note the dp-... namespace name
kubectl create secret generic chat-agent-openai \
  --namespace <dp-...-support-agent-development-...> \
  --from-literal=apiKey=sk-...
```

Then:

```bash
kubectl apply -f support-agent/components/chat-agent.yaml
kubectl apply -f support-agent/components/chat-gateway.yaml
```

## 4 — Verify the cross-project hop before going further

`chat-agent`'s Workload depends on `subscription-service` and `network-ops-service` — both in the
**telco-services** project — via `visibility: namespace`. This is supported by OpenChoreo's type
system and its generated `NetworkPolicy`/URL-resolution code, but it isn't exercised by any
existing sample in the main repo, so treat it as unproven until you've checked it here:

```bash
kubectl get pods -n <chat-agent's dp-... namespace>
kubectl exec -it deploy/chat-agent -n <chat-agent's dp-... namespace> -- \
  wget -qO- "$SUBSCRIPTION_SERVICE_URL/plans"
```

If that returns the seeded plan catalog as JSON, the hop works — move on. **If it hangs or gets
refused** (most likely cause: `support-agent` and `telco-services` don't share the same
org-level `Namespace` CR, or landed in different `Environment`s), the fallback is a one-line
change, already commented in `chat-agent.yaml`: flip both dependency entries' `visibility` to
`external`, flip the matching `visibility` on `subscription-service`/`network-ops-service`'s HTTP
endpoints to `external` too, and use their public URLs as literal `SUBSCRIPTION_SERVICE_URL` /
`NETWORK_OPS_SERVICE_URL` values instead of `envBindings`. It leaves the mesh and comes back in,
but it's guaranteed to work.

## 5 — Deploy Portal, then close the loop

```bash
kubectl apply -f portal/project.yaml
kubectl apply -f portal/components/customer-portal-ui.yaml
kubectl apply -f portal/components/employee-console-ui.yaml
```

Now find `chat-gateway`'s external URL (console, or `occ endpoint list --project support-agent
--component chat-gateway`), patch it into both `portal/components/*.yaml` files in place of
`CHANGE-ME-chat-gateway-external-url` (both the `http://` and `ws://` values), and re-apply:

```bash
kubectl apply -f portal/components/customer-portal-ui.yaml
kubectl apply -f portal/components/employee-console-ui.yaml
```

## 6 — Try it

Seeded demo customers (in `subscription-service`): `cust-001` Amara Perera, `cust-002` Nadeesha
Fernando, `cust-003` Kasun Silva, `cust-004` Ishara Jayawardena. Any non-empty employee id (e.g.
`agent-007`) works for the employee console — there's no real password on either login, this is a
demo.

**As a customer** (`customer-portal-ui`, log in as `cust-001`):
- *"What's my current plan?"*
- *"How much data did I use yesterday?"*
- *"Upgrade me to the Unlimited plan."* — then check `GET .../customers/cust-001/subscription` on
  subscription-service to see it actually changed, and `chat-db.audit_log` to see it was recorded.
- *"I keep losing signal near Nugegoda, can you report it?"*

**As an employee** (`employee-console-ui`, log in as `agent-007`, set "assisting customer" to
`cust-002`):
- *"Look up cust-003."*
- *"Give me the full account for cust-002."*
- *"Add a new plan: Streaming 50GB, 50GB, price 249900."*
- *"Resolve the open connectivity report for cust-001, note that a technician reset the tower."*

## Teardown

```bash
kubectl delete resource.openchoreo.dev chat-db telco-db -n default
kubectl delete project.openchoreo.dev portal support-agent telco-services -n default
```

Deleting the three `Project`s cascades to their `Component`s, `Workload`s, and cell namespaces.
