# Step 2 — Platform prerequisites

Before installing any OpenChoreo plane, we install the infrastructure the planes depend on:

- **Gateway API CRDs** and **kgateway** — gateway/ingress
- **cert-manager** — internal mTLS certificates
- **External Secrets Operator** + **OpenBao** — secret management (OpenBao is the backend; a
  `ClusterSecretStore` named `default` lets ExternalSecrets read from it)
- **CoreDNS rewrite** — resolves the `*.localhost` hostnames used by the console and gateways

## The fast path (recommended)

OpenChoreo ships a single script that installs all of the above and seeds OpenBao with the
development placeholder secrets the planes expect:

```bash
curl -fsSL "${RAW_BASE}/install/k3d/k3d-prerequisites.sh" | OPENCHOREO_REF="${OPENCHOREO_REF}" bash
```

This takes a few minutes (cert-manager, ESO, kgateway, and OpenBao each wait to become ready).

> **What gets seeded into OpenBao:** placeholder secrets for Backstage, the observer OAuth client,
> and **OpenSearch** credentials (`opensearch-username` / `opensearch-password`). On the
> `release-v1.1` line these seeds **do not** include OpenObserve credentials — we add those manually
> in [step 6](06-observability-plane-openobserve.md). Keep this in mind; it's the one place this
> workshop diverges from the default install.

## The manual path (optional — for understanding)

If you want to see each component, install them individually. Skip this section if you ran the script
above.

<details>
<summary>Expand manual prerequisite commands</summary>

### Gateway API CRDs

```bash
kubectl apply --server-side \
  -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.1/experimental-install.yaml
```

### cert-manager

```bash
helm upgrade --install cert-manager oci://quay.io/jetstack/charts/cert-manager \
  --namespace cert-manager --create-namespace \
  --version v1.19.4 \
  --set crds.enabled=true \
  --wait --timeout 180s
```

### External Secrets Operator

```bash
helm upgrade --install external-secrets oci://ghcr.io/external-secrets/charts/external-secrets \
  --namespace external-secrets --create-namespace \
  --version 2.0.1 \
  --set installCRDs=true \
  --wait --timeout 180s
```

### kgateway

```bash
helm upgrade --install kgateway-crds oci://cr.kgateway.dev/kgateway-dev/charts/kgateway-crds \
  --create-namespace --namespace openchoreo-control-plane \
  --version v2.2.1

helm upgrade --install kgateway oci://cr.kgateway.dev/kgateway-dev/charts/kgateway \
  --namespace openchoreo-control-plane --create-namespace \
  --version v2.2.1 \
  --set controller.extraEnv.KGW_ENABLE_GATEWAY_API_EXPERIMENTAL_FEATURES=true
```

### OpenBao

```bash
helm upgrade --install openbao oci://ghcr.io/openbao/charts/openbao \
  --namespace openbao --create-namespace \
  --version 0.25.6 \
  --values "${RAW_BASE}/install/k3d/common/values-openbao.yaml" \
  --wait --timeout 300s
```

### ClusterSecretStore

```bash
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-secrets-openbao
  namespace: openbao
---
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: default
spec:
  provider:
    vault:
      server: "http://openbao.openbao.svc:8200"
      path: "secret"
      version: "v2"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "openchoreo-secret-writer-role"
          serviceAccountRef:
            name: "external-secrets-openbao"
            namespace: "openbao"
EOF
```

### CoreDNS rewrite

```bash
kubectl apply -f "${RAW_BASE}/install/k3d/common/coredns-custom.yaml"
```

</details>

## Verify this step

```bash
# cert-manager, ESO, kgateway, OpenBao pods are Running
kubectl get pods -n cert-manager
kubectl get pods -n external-secrets
kubectl get pods -n openchoreo-control-plane -l app.kubernetes.io/name=kgateway
kubectl get pods -n openbao

# The default ClusterSecretStore is Valid
kubectl get clustersecretstore default
```

`kubectl get clustersecretstore default` should report `STATUS: Valid` and `READY: True`. If it
isn't ready, ESO can't reach OpenBao — check the `external-secrets` pods and the `openbao-0` pod
logs.

---

Next: [Step 3 — Control plane »](03-control-plane.md)
