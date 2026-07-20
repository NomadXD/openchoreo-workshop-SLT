# Step 6 — Observability plane (OpenObserve)

The observability plane collects **logs, traces, and metrics** from the other planes. The default
OpenChoreo install uses **OpenSearch** for logs and traces. In this workshop we swap those for
**[OpenObserve](https://openobserve.ai/)** — a lighter, Rust-based observability backend — while
keeping **Prometheus** for metrics.

This step is where the workshop diverges most from the stock install, so read the
[caveats](#caveats-on-v11) first.

## <a id="caveats-on-v11"></a>⚠️ Caveats on v1.1

We pin OpenChoreo to **v1.1.2 (GA)**. OpenObserve is a first-class, documented backend on the
OpenChoreo **v1.2** line; on v1.1 the situation is:

| What works | What to know |
|------------|--------------|
| OpenObserve installs and runs (logs `0.4.2`, tracing `0.2.4`, both `v1.0.x+` compatible) | These module versions predate the v1.2 console integration |
| Fluent Bit ships container logs into OpenObserve; the OTel collector ships traces | You view them in the **OpenObserve console**, not (necessarily) the OpenChoreo console |
| Prometheus metrics work as normal | The v1.1 control-plane *observer* service is wired for OpenSearch (`openSearchSecretName`), so the OpenChoreo console's **Runtime Logs** tab may show nothing for OpenObserve-backed logs |
| release-v1.1 seeds OpenSearch creds into OpenBao | It does **not** seed OpenObserve creds — we add them manually in [6.2](#62-seed-openobserve-credentials-into-openbao) |

**Bottom line for the workshop:** treat OpenObserve's own UI as the source of truth for logs/traces
here. If you need the console-integrated experience, install from the v1.2 line instead. Everything
below has been assembled from the OpenObserve module docs and the OpenChoreo install flow; the
observer↔OpenObserve console wiring on v1.1 is **not guaranteed** and is the one part to expect to
troubleshoot.

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

On `release-v1.1`, OpenBao is **not** seeded with OpenObserve credentials, so we add them. OpenBao
runs in dev mode with root token `root` on `127.0.0.1:8200` inside the pod:

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

Two sets of credentials are needed:

1. **`openobserve-admin-credentials`** — consumed by the OpenObserve modules (keys
   `ZO_ROOT_USER_EMAIL` / `ZO_ROOT_USER_PASSWORD`).
2. **`opensearch-admin-credentials`** and **`observer-secret`** — the v1.1 *observer* core service
   still mounts these at startup even though we're not running OpenSearch. Creating them keeps the
   observer pod from failing on a missing secret. (These read seeds that `release-v1.1` *does*
   provide.)

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
# --- OpenSearch admin credentials (mounted by the v1.1 observer core) ---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: opensearch-admin-credentials
  namespace: openchoreo-observability-plane
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: default
  target:
    name: opensearch-admin-credentials
  data:
  - secretKey: username
    remoteRef:
      key: opensearch-username
      property: value
  - secretKey: password
    remoteRef:
      key: opensearch-password
      property: value
---
# --- Observer service secret (v1.1 observer core) ---
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
  - secretKey: OPENSEARCH_USERNAME
    remoteRef:
      key: opensearch-username
      property: value
  - secretKey: OPENSEARCH_PASSWORD
    remoteRef:
      key: opensearch-password
      property: value
  - secretKey: UID_RESOLVER_OAUTH_CLIENT_SECRET
    remoteRef:
      key: observer-oauth-client-secret
      property: value
EOF

kubectl wait -n openchoreo-observability-plane \
  --for=condition=Ready \
  externalsecret/openobserve-admin-credentials \
  externalsecret/opensearch-admin-credentials \
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
front of the pluggable logs/traces/metrics modules:

```bash
helm upgrade --install openchoreo-observability-plane oci://ghcr.io/openchoreo/helm-charts/openchoreo-observability-plane \
  --version "${OPENCHOREO_VERSION}" \
  --namespace openchoreo-observability-plane \
  --values "${RAW_BASE}/install/k3d/single-cluster/values-op.yaml" \
  --timeout 25m
```

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
  --version 0.4.2
```

Then enable **log collection** — this turns on the Fluent Bit DaemonSet that tails container logs
and ships them to OpenObserve:

```bash
helm upgrade observability-logs-openobserve \
  oci://ghcr.io/openchoreo/helm-charts/observability-logs-openobserve \
  --namespace openchoreo-observability-plane \
  --version 0.4.2 \
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
> - *Observer pod not starting* — it mounts `observer-secret` / `opensearch-admin-credentials`;
>   confirm those ExternalSecrets synced in [6.3](#63-create-the-externalsecrets).

---

Next: [Step 7 — Verify »](07-verify.md)
