# Step 6 — Observability plane (OpenObserve)

The observability plane collects **logs, traces, and metrics** from the other planes. The stock
OpenChoreo v1.2 install uses **OpenSearch** for logs and traces. In this workshop we swap those for
**[OpenObserve](https://openobserve.ai/)** — a lighter, Rust-based observability backend — while
keeping **Prometheus** for metrics.

## How this differs from the stock install

On the **v1.2** line OpenObserve is a first-class, documented backend, so this swap is
straightforward:

| What we change | How it works |
|----------------|--------------|
| Install the OpenObserve **logs** (`0.5.1`) and **tracing** (`0.2.4`) modules instead of the OpenSearch ones | Each module ships an *adapter* service the observer queries (`logs-adapter`, `tracing-adapter`) |
| Point the observer at the OpenObserve module adapters (default `observer.logsAdapter.url` / `observer.tracingAdapter.url`, see [6.5](#65-install-the-observability-plane-core)) | The **OpenChoreo console**'s Runtime Logs/Traces tabs read OpenObserve data through those adapters — no OpenSearch wiring involved |
| Seed OpenObserve admin credentials into OpenBao ([6.2](#62-seed-openobserve-credentials-into-openbao)) | The default install only seeds OpenSearch creds, so we add the OpenObserve ones ourselves |
| Fluent Bit ships container logs into OpenObserve; the tracing module receives traces | Prometheus metrics work exactly as in the stock install |

You can view logs and traces in **either** the OpenObserve console (see [the end of this
step](#open-the-openobserve-console)) **or** the OpenChoreo console's Runtime Logs/Traces tabs.

## 6.1 Create the namespace and trust the control-plane CA

```bash
kubectl create namespace openchoreo-observability-plane --dry-run=client -o yaml | kubectl apply -f -

CA_CRT=$(kubectl get secret cluster-gateway-ca \
  -n openchoreo-control-plane -o jsonpath='{.data.ca\.crt}' | base64 -d)

kubectl create configmap cluster-gateway-ca \
  --from-literal=ca.crt="$CA_CRT" \
  -n openchoreo-observability-plane --dry-run=client -o yaml | kubectl apply -f -
```

## 6.2 Seed OpenObserve credentials into OpenBao

The stock (OpenSearch-based) install does **not** seed OpenObserve credentials into OpenBao, so we
add them. OpenBao runs in dev mode with root token `root` on `127.0.0.1:8200` inside the pod:

```bash
kubectl exec -n openbao openbao-0 -- sh -c '
  export BAO_ADDR=http://127.0.0.1:8200
  export BAO_TOKEN=root
  bao kv put secret/openobserve-admin-email    value="admin@openchoreo.localhost"
  bao kv put secret/openobserve-admin-password value="ThisIsTheOpenObservePassword1"
'
```

> These are the same placeholder values the OpenChoreo v1.2 install uses, so downstream defaults line
> up. Note them down — you'll log into the OpenObserve console with
> `admin@openchoreo.localhost` / `ThisIsTheOpenObservePassword1`.

## 6.3 Create the ExternalSecrets

Two credentials are needed:

1. **`openobserve-admin-credentials`** — consumed by the OpenObserve modules (keys
   `ZO_ROOT_USER_EMAIL` / `ZO_ROOT_USER_PASSWORD`), including the single-node OpenObserve the logs
   module brings up.
2. **`observer-secret`** — referenced by the observability-plane core chart
   (`observer.secretName`). On v1.2 the observer reaches logs and traces through the module adapter
   services, so the only key it needs here is the UID resolver's OAuth client secret.

```bash
kubectl apply -f - <<'EOF'
# --- OpenObserve admin credentials (for the OpenObserve modules) ---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: openobserve-admin-credentials
  namespace: openchoreo-observability-plane
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: default
  target:
    name: openobserve-admin-credentials
  data:
  - secretKey: ZO_ROOT_USER_EMAIL
    remoteRef:
      key: openobserve-admin-email
      property: value
  - secretKey: ZO_ROOT_USER_PASSWORD
    remoteRef:
      key: openobserve-admin-password
      property: value
---
# --- Observer service secret (observability-plane core) ---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: observer-secret
  namespace: openchoreo-observability-plane
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: default
  target:
    name: observer-secret
  data:
  - secretKey: UID_RESOLVER_OAUTH_CLIENT_SECRET
    remoteRef:
      key: observer-oauth-client-secret
      property: value
EOF

kubectl wait -n openchoreo-observability-plane \
  --for=condition=Ready \
  externalsecret/openobserve-admin-credentials \
  externalsecret/observer-secret --timeout=60s
```

## 6.4 Generate a machine ID

Some observability components require a populated `/etc/machine-id` on the node:

```bash
docker exec k3d-openchoreo-server-0 sh -c \
  "cat /proc/sys/kernel/random/uuid | tr -d '-' > /etc/machine-id"
```

## 6.5 Install the observability-plane core

This installs the observer API, RCA/FinOps agents, and the gateway — the shared services that sit in
front of the pluggable logs/traces/metrics modules. On v1.2 the observer's **logs and tracing
adapters** are wired by default (via `observer.logsAdapter.url` / `observer.tracingAdapter.url`,
which point at the `logs-adapter` / `tracing-adapter` services the OpenObserve modules ship), so no
extra flags are needed here:

```bash
helm upgrade --install openchoreo-observability-plane oci://ghcr.io/openchoreo/helm-charts/openchoreo-observability-plane \
  --version "${OPENCHOREO_VERSION}" \
  --namespace openchoreo-observability-plane \
  --values "${RAW_BASE}/install/k3d/single-cluster/values-op.yaml" \
  --timeout 25m
```

> **On chart `1.2.0-rc.1` the adapters have no `enabled` flag** — earlier drafts of this guide passed
> `--set observer.logsAdapter.enabled=true` / `--set observer.tracingAdapter.enabled=true`, but the
> chart's values schema rejects those keys (it only exposes `url` / `timeout`). The observer always
> queries the default `logs-adapter:9098` / `tracing-adapter:9100` URLs, which is exactly what wires
> OpenObserve into the OpenChoreo console. Just install the OpenObserve modules
> ([6.7](#67-install-the-openobserve-logs-module), [6.8](#68-install-the-openobserve-tracing-module))
> so those services exist before expecting the console's Runtime Logs/Traces tabs to return data.

## 6.6 Install the metrics module (Prometheus)

```bash
helm upgrade --install observability-metrics-prometheus \
  oci://ghcr.io/openchoreo/helm-charts/observability-metrics-prometheus \
  --create-namespace \
  --namespace openchoreo-observability-plane \
  --version 0.6.1
```

## 6.7 Install the OpenObserve **logs** module

Install in standalone mode first (this also deploys a single-node OpenObserve that the tracing
module will reuse):

```bash
helm upgrade --install observability-logs-openobserve \
  oci://ghcr.io/openchoreo/helm-charts/observability-logs-openobserve \
  --create-namespace \
  --namespace openchoreo-observability-plane \
  --version 0.5.1
```

Then enable **log collection** — this turns on the Fluent Bit DaemonSet that tails container logs
and ships them to OpenObserve:

```bash
helm upgrade observability-logs-openobserve \
  oci://ghcr.io/openchoreo/helm-charts/observability-logs-openobserve \
  --namespace openchoreo-observability-plane \
  --version 0.5.1 \
  --reuse-values \
  --set fluent-bit.enabled=true
```

## 6.8 Install the OpenObserve **tracing** module

The tracing module can reuse the OpenObserve instance the logs module already deployed, so we disable
its bundled standalone OpenObserve:

```bash
helm upgrade --install observability-tracing-openobserve \
  oci://ghcr.io/openchoreo/helm-charts/observability-tracing-openobserve \
  --create-namespace \
  --namespace openchoreo-observability-plane \
  --version 0.2.4 \
  --set openobserve-standalone.enabled=false
```

## 6.9 Register the observability plane with the control plane

```bash
kubectl wait -n openchoreo-observability-plane \
  --for=jsonpath='{.data.ca\.crt}' secret/cluster-agent-tls --timeout=120s

AGENT_CA=$(kubectl get secret cluster-agent-tls \
  -n openchoreo-observability-plane -o jsonpath='{.data.ca\.crt}' | base64 -d)

kubectl apply -f - <<EOF
apiVersion: openchoreo.dev/v1alpha1
kind: ClusterObservabilityPlane
metadata:
  name: default
spec:
  planeID: default
  clusterAgent:
    clientCA:
      value: |
$(echo "$AGENT_CA" | sed 's/^/        /')
  observerURL: http://observer.openchoreo.localhost:11080
EOF
```

## 6.10 Link the observability plane to the other planes

```bash
kubectl patch clusterdataplane default --type merge \
  -p '{"spec":{"observabilityPlaneRef":{"kind":"ClusterObservabilityPlane","name":"default"}}}'

kubectl patch clusterworkflowplane default --type merge \
  -p '{"spec":{"observabilityPlaneRef":{"kind":"ClusterObservabilityPlane","name":"default"}}}'
```

## Verify this step

```bash
# All observability-plane pods (observer, OpenObserve, Prometheus, Fluent Bit, OTel collector)
kubectl get pods -n openchoreo-observability-plane

# Fluent Bit is running on every node (DaemonSet)
kubectl get daemonset -n openchoreo-observability-plane

# OpenObserve is up
kubectl get pods -n openchoreo-observability-plane -l app.kubernetes.io/name=openobserve

# Control plane sees the observability plane
kubectl get clusterobservabilityplane default
```

### Open the OpenObserve console

Port-forward the OpenObserve service and log in with the credentials from
[6.2](#62-seed-openobserve-credentials-into-openbao):

```bash
# Service name may be openobserve-standalone or observability-logs-openobserve-openobserve-standalone;
# list services to find it:
kubectl get svc -n openchoreo-observability-plane | grep -i openobserve

# Port-forward (adjust the service name to what the previous command shows)
kubectl port-forward -n openchoreo-observability-plane svc/observability-logs-openobserve-openobserve-standalone 5080:5080
```

Then open <http://localhost:5080> and log in as:

- **Email:** `admin@openchoreo.localhost`
- **Password:** `ThisIsTheOpenObservePassword1`

Once you deploy a component in [step 7](07-verify.md), its container logs will appear under the log
streams here, and traces (for instrumented services) under the traces view.

> **Troubleshooting**
> - *OpenObserve pod `CrashLoopBackOff` with an auth error* — the `openobserve-admin-credentials`
>   secret is missing or wrong. Re-run [6.2](#62-seed-openobserve-credentials-into-openbao) and
>   confirm the ExternalSecret in [6.3](#63-create-the-externalsecrets) is `Ready`.
> - *No logs appearing* — check the Fluent Bit DaemonSet is `Running` on all nodes and that
>   `fluent-bit.enabled=true` from [6.7](#67-install-the-openobserve-logs-module) actually rolled
>   out (`helm get values observability-logs-openobserve -n openchoreo-observability-plane`).
> - *Observer pod not starting* — it mounts `observer-secret`; confirm that ExternalSecret synced in
>   [6.3](#63-create-the-externalsecrets).
> - *OpenObserve data missing in the OpenChoreo console* — confirm the module `logs-adapter` /
>   `tracing-adapter` services and pods are `Running` in `openchoreo-observability-plane`, and that
>   the observer's default adapter URLs still match them (`kubectl get svc -n
>   openchoreo-observability-plane | grep -iE 'logs-adapter|tracing-adapter'` should show
>   `logs-adapter:9098` / `tracing-adapter:9100`). The OpenObserve console (below) always works
>   regardless.

---

Next: [Step 7 — Verify »](07-verify.md)
