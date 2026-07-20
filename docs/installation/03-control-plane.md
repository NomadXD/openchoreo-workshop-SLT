# Step 3 — Control plane

The control plane hosts the OpenChoreo API, the controllers, and the Backstage-based developer
console. It also depends on **Thunder** (the identity provider) for console login.

## 3.1 Install Thunder (identity provider)

```bash
helm upgrade --install thunder oci://ghcr.io/asgardeo/helm-charts/thunder \
  --namespace thunder --create-namespace \
  --version 0.28.0 \
  --values "${RAW_BASE}/install/k3d/common/values-thunder.yaml"

kubectl wait -n thunder \
  --for=condition=available --timeout=300s deployment -l app.kubernetes.io/name=thunder
```

## 3.2 Create the Backstage secrets

The console reads its backend/client secrets from OpenBao via an ExternalSecret:

```bash
kubectl apply -f - <<'EOF'
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: backstage-secrets
  namespace: openchoreo-control-plane
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: ClusterSecretStore
    name: default
  target:
    name: backstage-secrets
  data:
  - secretKey: backend-secret
    remoteRef:
      key: backstage-backend-secret
      property: value
  - secretKey: client-secret
    remoteRef:
      key: backstage-client-secret
      property: value
  - secretKey: jenkins-api-key
    remoteRef:
      key: backstage-jenkins-api-key
      property: value
EOF
```

## 3.3 Install the control plane

```bash
helm upgrade --install openchoreo-control-plane oci://ghcr.io/openchoreo/helm-charts/openchoreo-control-plane \
  --version "${OPENCHOREO_VERSION}" \
  --namespace openchoreo-control-plane --create-namespace \
  --values "${RAW_BASE}/install/k3d/single-cluster/values-cp.yaml"

kubectl wait -n openchoreo-control-plane \
  --for=condition=available --timeout=300s deployment --all
```

## 3.4 Install the default platform resources

These create the default Organization, DeploymentPipeline, and related resources OpenChoreo needs
to accept components:

```bash
kubectl label namespace default openchoreo.dev/control-plane=true --overwrite

kubectl apply -f "${RAW_BASE}/samples/getting-started/all.yaml"
```

## Verify this step

```bash
# All control-plane deployments available
kubectl get deploy -n openchoreo-control-plane

# The API/controller manager is up
kubectl get pods -n openchoreo-control-plane

# CRDs are installed
kubectl get crd | grep openchoreo.dev | head
```

Every deployment in `openchoreo-control-plane` should show `AVAILABLE 1` (or more). The
`cluster-gateway-ca` certificate (used to secure plane-to-plane traffic) should also exist — the
data/workflow/observability plane steps depend on it:

```bash
kubectl get certificate cluster-gateway-ca -n openchoreo-control-plane
```

> **Troubleshooting**
> - *Backstage pod not starting* — check that `kubectl get externalsecret backstage-secrets -n
>   openchoreo-control-plane` reports `SecretSynced=True`. If not, OpenBao/ESO from step 2 isn't
>   healthy.
> - *Login later fails* — Thunder (3.1) must be `available` before the console can complete OIDC.

---

Next: [Step 4 — Data plane »](04-data-plane.md)
