# Telco Chat Demo — Console (Build from Source) Walkthrough

This is the **console-driven** path through the Vantage Mobile telco chat demo — creating every
Project/Resource/Component by hand in the OpenChoreo developer console, using **Build from
Source** against the real service code in this repo, rather than applying the pre-written YAML in
[`../README.md`](README.md).

Use this doc if you want to *see* the console mechanics work (build-from-source, resource
provisioning, cross-project dependency wiring) rather than just `kubectl apply` the finished
manifests. It's written to be run from scratch on a clean environment — follow it in order.

> If anything here disagrees with what the console actually shows you, that's useful information —
> tell me and we'll fix the guide or the underlying config.

---

## Before you start

- A running OpenChoreo environment (all four planes) — see the
  [installation guide](../../installation/README.md). Confirm the console loads and the
  workflow plane is up:
  ```bash
  kubectl config current-context        # should be k3d-openchoreo
  kubectl get clusterworkflowplane      # should list `default`
  ```
- Console: **http://openchoreo.localhost:8080** — `admin@openchoreo.dev` / `Admin@123`.
- The demo's source code, pushed to a branch this cluster's workflow plane can clone:
  - **Repo:** `https://github.com/NomadXD/openchoreo-workshop-SLT`
  - **Branch:** `telco-chat-demo`
  - (If you're working from a fork or a different branch, substitute your own values everywhere
    below — the Application Path / Docker Context / Dockerfile Path are all relative to *this*
    repo's root, i.e. `docs/scenarios/telco-chat-demo/services/<name>`.)

### Architecture recap

Three OpenChoreo projects — **Telco Services** (system of record), **Support Agent** (the chat
layer), **Portal** (frontends only). Full design, diagrams, and the plain-`kubectl` deployment path
are in [`../README.md`](README.md). This guide builds the same six components, just through the
console instead.

### Progress tracker

| # | Component | Project | Status |
|---|---|---|---|
| 1 | `telco-db` (Postgres `Resource`) | `default` (standing in for `telco-services`) | ✅ done |
| 2 | `subscription-service` | `default` | ✅ done |
| 3 | `network-ops-service` | `default` | ✅ done |
| 4 | `support-agent` project | — | ✅ created |
| 5 | `chat-agent` | `support-agent` | ✅ done (cross-project dependency proven working) |
| 6 | `chat-cache` (Valkey `Resource`) | `support-agent` | ⬜ next |
| 7 | `chat-db` (Postgres `Resource`) | `support-agent` | ⬜ next |
| 8 | `chat-gateway` | `support-agent` | ⬜ next |
| 9 | `customer-portal-ui` | `portal` | ⬜ next |
| 10 | `employee-console-ui` | `portal` | ⬜ next |

> **Note on projects:** we used the pre-existing `default` project for `telco-db` /
> `subscription-service` / `network-ops-service` instead of creating a separate `telco-services`
> project first. That's fine for proving the mechanics — everything below still demonstrates real
> cross-project wiring (`default` → `support-agent`). If you want the "real" three-project layout
> from the design doc, create a `telco-services` project the same way `support-agent` was created
> (§2.1) before Part 1, and use it in place of `default` throughout.

---

## Part 1 — Telco Services (done)

### 1.1 Create the `telco-db` Postgres resource

1. Sidebar → **Create…** → **Resource** → **Postgres**.
2. **Namespace** / **Project**: leave both `default`.
3. **Resource Name:** `telco-db`. Click **Next**.
4. **Postgres Details** → **database** = `telco`. **Review** → **Create**.

### 1.2 Deploy it

Creating a resource only registers it — it isn't running yet:

5. Open `telco-db` → **DEPLOY** tab → **Set up** → **Configure & Deploy** → **Next** → **Deploy**.
6. Wait for **Development** to show **Active**.

**Known hiccup:** the bundled Adminer sidecar pod sometimes hits a transient Docker Hub
`500` pulling its image and briefly shows `ImagePullBackOff`. It resolves on its own within
a minute or two — it isn't the actual database, which comes up fine independently. If the
resource is still stuck "Pending" after a couple of minutes, check:
```bash
kubectl get pods -n <its dp-...-development-... namespace> -o wide
```

### 1.3 Verify

```bash
kubectl get resource.openchoreo.dev telco-db -n default
```

### 1.4 Create `subscription-service`

**Create…** → **Component** → **Service** template.

**Component Metadata:**
- Namespace/Project: `default`
- **Component Name:** `subscription-service`
- **Description:** `Vantage Mobile's BSS — accounts, plan catalog, subscriptions`

**Build & Deploy:**

| Field | Value |
|---|---|
| Deployment source | **Build from Source** |
| Build Workflow | `dockerfile-builder` |
| Git Repository URL | `https://github.com/NomadXD/openchoreo-workshop-SLT` |
| Branch | `telco-chat-demo` |
| Application Path | `docs/scenarios/telco-chat-demo/services/subscription-service` |
| Docker Context | `/docs/scenarios/telco-chat-demo/services/subscription-service` |
| Dockerfile Path | `/docs/scenarios/telco-chat-demo/services/subscription-service/Dockerfile` |

**Endpoint:** `subscription-api` — HTTP, port `8080`, visibility **External**.

**Resource Dependency:** **Add Resource Dependency** → `telco-db` → bind output `url` → env var
`DATABASE_URL`.

**Review** → **Create** → open it → **DEPLOY** → **Set up** → **Configure & Deploy** → **Next** →
**Deploy**. Build takes a few minutes — wait for **Active**.

**Verify:**
```bash
BASE=<subscription-service's external URL — find it via Try Out, or:
  kubectl get releasebinding.openchoreo.dev subscription-service-development -n default \
    -o jsonpath='{.status.endpoints[0].externalURLs.http}'>
curl -s "$BASE/healthz"                              # {"status":"ok"}
curl -s "$BASE/plans" | python3 -m json.tool          # 3 seeded plans
curl -s "$BASE/customers" | python3 -m json.tool      # 4 seeded demo customers
```

### 1.5 Create `network-ops-service`

Same shape, same project, same resource:

**Component Metadata:**
- **Component Name:** `network-ops-service`
- **Description:** `Vantage Mobile's OSS — usage, service-disruption reports`

**Build & Deploy:**

| Field | Value |
|---|---|
| Deployment source | **Build from Source** |
| Build Workflow | `dockerfile-builder` |
| Git Repository URL | `https://github.com/NomadXD/openchoreo-workshop-SLT` |
| Branch | `telco-chat-demo` |
| Application Path | `docs/scenarios/telco-chat-demo/services/network-ops-service` |
| Docker Context | `/docs/scenarios/telco-chat-demo/services/network-ops-service` |
| Dockerfile Path | `/docs/scenarios/telco-chat-demo/services/network-ops-service/Dockerfile` |

**Endpoint:** `network-ops-api` — HTTP, port `8080`, visibility **External**.

**Resource Dependency:** `telco-db` → bind output `url` → env var `DATABASE_URL`.

Review → Create → Deploy → wait for Active.

> ⚠️ **Watch the Docker Context / Dockerfile Path fields carefully.** We hit a real build failure
> here from a stray character: the console recorded `. /docs/scenarios/...` (leading `.` + space)
> instead of `/docs/scenarios/...`, which fails with `Docker build context directory not found`.
> Clear the field completely before retyping — don't just append/edit over a pre-filled default.

**Verify:**
```bash
BASE=<its external URL>
curl -s "$BASE/healthz"
curl -s "$BASE/reports?status=open" | python3 -m json.tool   # cust-001's seeded connectivity report
```

---

## Part 2 — Enabling cross-project access (done)

This is the part that actually matters: `chat-agent` will live in a **different project**
(`support-agent`) than `subscription-service`/`network-ops-service` (`default`), and needs to reach
them via `namespace` visibility — the mechanism flagged as unverified-by-any-sample in the design
doc. **It works.** Here's exactly how we proved it, including the two real gotchas we hit.

### 2.1 Create the `support-agent` project

The console's Create wizard didn't have an obvious "New Project" flow surfaced to us, so we
applied the Project directly:

```bash
kubectl apply -f /path/to/openchoreo-workshop-SLT/docs/scenarios/telco-chat-demo/support-agent/project.yaml
```

(If your console does have a project-creation flow, use it instead with **Project Name:**
`support-agent` — same effect.)

### 2.2 Add `namespace` visibility to both telco-services endpoints

Open `subscription-service` (and separately `network-ops-service`) in the console → find the
endpoint config → tick **Namespace** visibility alongside the existing **External** (same
checkbox group as External/Internal/Project from scenario 03) → save → **Configure & Deploy** →
**Deploy** again.

### 2.3 ⚠️ The gotcha: a redeploy doesn't always promote itself

This is the one thing in this whole guide worth reading twice. OpenChoreo's render pipeline is:

```
Component + Workload  →  ComponentRelease (immutable snapshot)  →  ReleaseBinding  →  rendered K8s objects
```

Changing an endpoint's visibility and clicking Deploy creates a **new** `ComponentRelease` — but
we saw the `ReleaseBinding` stay pinned to the **old** release, meaning the actual deployed
`NetworkPolicy` never picked up the new visibility rule, even though the console showed the
component as re-deployed successfully. Symptom: same-project calls work fine, but the
cross-project call gets a hard, fast `Connection refused` (not a timeout).

**Check whether this happened to you:**
```bash
# Does the newest ComponentRelease actually have the visibility you expect?
kubectl get componentrelease.openchoreo.dev -n default | grep subscription-service
kubectl get componentrelease.openchoreo.dev/<newest-one> -n default \
  -o jsonpath='{.spec.workload.endpoints}'

# Is the ReleaseBinding actually pointing at that newest release?
kubectl get releasebinding.openchoreo.dev subscription-service-development -n default \
  -o jsonpath='{.spec.releaseName}'
```

If `releaseName` still names the *older* release, that's the bug. Fix (until we find the proper
console/`occ` action for this — see **Open question** below):

```bash
kubectl patch releasebinding.openchoreo.dev subscription-service-development -n default \
  --type=merge -p='{"spec":{"releaseName":"<the newest ComponentRelease name>"}}'
```

Re-check the `NetworkPolicy` afterward — it should gain a third ingress rule shaped like:
```yaml
- from:
    - namespaceSelector:
        matchLabels:
          openchoreo.dev/namespace: default
          openchoreo.dev/environment: development
```

**Open question, not yet resolved:** in our run, `subscription-service` needed this manual patch,
but `network-ops-service`'s `ReleaseBinding` picked up its new release on its own a little later —
so this might just be a timing race (auto-promotion eventually happens, we didn't wait long
enough) rather than something that's always stuck. **If you hit the "Connection refused" symptom,
wait 1–2 minutes and re-check before patching anything** — only patch if it's still stale.

### 2.4 Create `chat-agent` in `support-agent`

**Create…** → **Component** → **Service**:
- **Namespace:** `default` / **Project:** `support-agent`
- **Component Name:** `chat-agent`
- **Description:** `Role-scoped LangChain tool router`

**Build & Deploy:**

| Field | Value |
|---|---|
| Deployment source | **Build from Source** |
| Build Workflow | `dockerfile-builder` |
| Git Repository URL | `https://github.com/NomadXD/openchoreo-workshop-SLT` |
| Branch | `telco-chat-demo` |
| Application Path | `docs/scenarios/telco-chat-demo/services/chat-agent` |
| Docker Context | `/docs/scenarios/telco-chat-demo/services/chat-agent` |
| Dockerfile Path | `/docs/scenarios/telco-chat-demo/services/chat-agent/Dockerfile` |

**Endpoint:** `http` — HTTP, port `8080`, visibility **Project** only.

**Environment variable:** `OPENAI_API_KEY` — see the security note in Part 3 before you set this
for real; a placeholder string is enough to pass chat-agent's own startup check for this test.

**Component Dependency** (the thing we were actually testing): **Add Component Dependency** twice —
- `subscription-service` → endpoint `subscription-api` → visibility **Namespace** → bind
  `address` → `SUBSCRIPTION_SERVICE_URL`
- `network-ops-service` → endpoint `network-ops-api` → visibility **Namespace** → bind
  `address` → `NETWORK_OPS_SERVICE_URL`

**The console's picker did successfully offer components from a different project** (`default`,
while creating this component in `support-agent`) — that was the open question from the design
doc, now answered: yes, it works.

Review → Create → Deploy → wait for Active.

### 2.5 Verify cross-project connectivity

```bash
support_ns=$(kubectl get ns -o name | grep "dp-default-support-agent-development" | sed 's|namespace/||')
pod=$(kubectl get pod -n "$support_ns" -o name | grep chat-agent | head -1 | sed 's|pod/||')

kubectl exec -n "$support_ns" "$pod" -- python3 -c "
import urllib.request
print(urllib.request.urlopen('http://subscription-service.<telco-services-dp-namespace>.svc.cluster.local:8080/plans', timeout=5).read().decode())
"
```

(Find `<telco-services-dp-namespace>` with `kubectl get ns | grep dp-default-default-development`,
or read it straight off `chat-agent`'s own resolved env var: `kubectl exec ... -- env | grep
SUBSCRIPTION_SERVICE_URL`.)

**Expected:** the JSON plan catalog comes back. If you get `Connection refused`, go back to §2.3.

---

## Part 3 — Before this is demo-safe

- **Rotate the OpenAI key used during testing**, and stop passing it as a plaintext literal env
  var (`kubectl get workload -n support-agent -o yaml` shows it in cleartext to anyone with cluster
  read access — same for the `ComponentRelease` snapshot). Use a real Kubernetes Secret instead:
  ```bash
  kubectl create secret generic chat-agent-openai \
    --namespace <chat-agent's dp-...-support-agent-development-... namespace> \
    --from-literal=apiKey=sk-...
  ```
  then point the Workload's env var at `valueFrom.secretKeyRef.{name: chat-agent-openai, key:
  apiKey}` instead of a literal value (this is exactly how `support-agent/components/chat-agent.yaml`
  in the sibling kubectl path already does it).

---

## Part 4 — Next steps (not yet done)

### 4.1 `chat-db` and `chat-cache` (both `Resource`s)

Same flow as `telco-db` in §1.1–1.2, twice:

**`chat-db`** (Postgres):
- **Project:** `support-agent`
- **Resource Name:** `chat-db`
- **database:** `supportagent`

**`chat-cache`** (Valkey — Redis-protocol-compatible):
- **Project:** `support-agent`
- **Resource Name:** `chat-cache`
- No configurable parameters beyond the defaults (`memory`, `adminEnabled`) — leave them.

Deploy both the same way (**DEPLOY** → **Set up** → **Configure & Deploy** → **Deploy**, wait for
Active). Note: `valkey` always runs with a generated password — there's no toggle to disable
auth — so `chat-gateway` binds its `url` output (a full `redis://:<password>@host:6379` string),
not a bare host:port.

### 4.2 `chat-gateway`

**Namespace/Project:** `support-agent`. Build from Source, same repo/branch:
- Application Path: `docs/scenarios/telco-chat-demo/services/chat-gateway`
- Docker Context: `/docs/scenarios/telco-chat-demo/services/chat-gateway`
- Dockerfile Path: `/docs/scenarios/telco-chat-demo/services/chat-gateway/Dockerfile`
- **Endpoint:** `http` — HTTP, port `8080`, visibility **External**
- **Env var:** `JWT_SECRET` (pick a real value, don't leave the code's dev default in place for
  anything beyond this test)
- **Component Dependency:** `chat-agent` → endpoint `http` → visibility **Project** → bind
  `address` → `CHAT_AGENT_URL`
- **Resource Dependency:** `chat-db` → bind output `url` → env var `DATABASE_URL`
- **Resource Dependency:** `chat-cache` → bind output `url` → env var `REDIS_URL`

Same rule as §2.3 applies here — after deploying, double-check `releasebinding.spec.releaseName`
actually points at the release you think it does before trusting anything works.

**Verify:**
```bash
BASE=<chat-gateway's external URL>
curl -s -X POST "$BASE/api/auth/customer/login" -H "Content-Type: application/json" \
  -d '{"customerId":"cust-001"}'
# should return {"token": "..."}
```

### 4.3 Frontends: `customer-portal-ui` / `employee-console-ui`

**Namespace/Project:** `portal` (create this project first, same as §2.1 if the console still
doesn't expose project creation). Build from Source:

| | `customer-portal-ui` | `employee-console-ui` |
|---|---|---|
| Application Path | `docs/scenarios/telco-chat-demo/services/customer-portal-ui` | `docs/scenarios/telco-chat-demo/services/employee-console-ui` |
| Docker Context | `/docs/scenarios/telco-chat-demo/services/customer-portal-ui` | `/docs/scenarios/telco-chat-demo/services/employee-console-ui` |
| Dockerfile Path | `.../customer-portal-ui/Dockerfile` | `.../employee-console-ui/Dockerfile` |
| Endpoint | HTTP `8080`, External | HTTP `8080`, External |

**Neither of these can use a Component Dependency for `chat-gateway`'s URL** — it's a public URL a
different project's *browser* needs, not an internal service-to-service call, and that mechanism
only injects internal `project`/`namespace`-visible addresses. Instead: deploy `chat-gateway`
first, copy its external URL, and set these two literal env vars on each frontend:
- `CHAT_GATEWAY_HTTP_URL` = `http://<chat-gateway's external host>`
- `CHAT_GATEWAY_WS_URL` = `ws://<chat-gateway's external host>`

### 4.4 Full demo script

Once everything above is Active, seeded demo customers are `cust-001` Amara Perera, `cust-002`
Nadeesha Fernando, `cust-003` Kasun Silva, `cust-004` Ishara Jayawardena; any non-empty employee id
(e.g. `agent-007`) works for the console.

**As a customer** (log in as `cust-001` on `customer-portal-ui`):
- *"What's my current plan?"*
- *"How much data did I use yesterday?"*
- *"Upgrade me to the Unlimited plan."*
- *"I keep losing signal near Nugegoda, can you report it?"*

**As an employee** (log in as `agent-007` on `employee-console-ui`, assisting customer `cust-002`):
- *"Look up cust-003."*
- *"Give me the full account for cust-002."*
- *"Add a new plan: Streaming 50GB, 50GB, price 249900."*
- *"Resolve the open connectivity report for cust-001."*

---

## Appendix — quick troubleshooting reference

| Symptom | Cause | Fix |
|---|---|---|
| Build fails: `Docker build context directory not found` | Stray character (leading `.`/space) in Docker Context or Dockerfile Path field | Clear the field fully, retype exactly `/docs/scenarios/telco-chat-demo/services/<name>` |
| Resource stuck "Pending" briefly after creation | Bundled Adminer sidecar hit a transient Docker Hub `500` | Wait 1–2 min, refresh the console; check `kubectl get pods -n <dp-... namespace>` |
| Cross-project call gets fast `Connection refused` (not a timeout) | `ReleaseBinding.spec.releaseName` still pinned to a release from before the visibility change | Confirm with `kubectl get componentrelease` + `kubectl get releasebinding -o jsonpath='{.spec.releaseName}'`; patch if genuinely stale (§2.3) |
| Same-project call works, cross-project doesn't | Missing `namespace` visibility on the producer's endpoint, or the promotion gotcha above | Check the endpoint's `visibility` list includes `namespace`, then check the promotion |
