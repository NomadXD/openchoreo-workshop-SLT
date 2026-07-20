# Step 4 — Data plane

The data plane is where your components run. It registers with the control plane through a
**cluster agent** (mTLS), so we first hand it the control plane's CA, install the plane, then
register it as a `ClusterDataPlane`.

## 4.1 Create the namespace and trust the control-plane CA

```bash
kubectl create namespace openchoreo-data-plane --dry-run=client -o yaml | kubectl apply -f -

# Wait for the CA the control plane issued in step 3
kubectl wait -n openchoreo-control-plane \
  --for=condition=Ready certificate/cluster-gateway-ca --timeout=120s

# Copy the CA into the data-plane namespace so the cluster agent trusts the gateway
CA_CRT=$(kubectl get secret cluster-gateway-ca \
  -n openchoreo-control-plane -o jsonpath='{.data.ca\.crt}' | base64 -d)

kubectl create configmap cluster-gateway-ca \
  --from-literal=ca.crt="$CA_CRT" \
  -n openchoreo-data-plane --dry-run=client -o yaml | kubectl apply -f -
```

## 4.2 Install the data plane

```bash
helm upgrade --install openchoreo-data-plane oci://ghcr.io/openchoreo/helm-charts/openchoreo-data-plane \
  --version "${OPENCHOREO_VERSION}" \
  --namespace openchoreo-data-plane --create-namespace \
  --values "${RAW_BASE}/install/k3d/single-cluster/values-dp.yaml"
```

## 4.3 Register the data plane with the control plane

The control plane needs the data plane's agent CA to establish the mTLS tunnel. We read it from the
`cluster-agent-tls` secret and create a `ClusterDataPlane` resource:

```bash
kubectl wait -n openchoreo-data-plane \
  --for=jsonpath='{.data.ca\.crt}' secret/cluster-agent-tls --timeout=120s

AGENT_CA=$(kubectl get secret cluster-agent-tls \
  -n openchoreo-data-plane -o jsonpath='{.data.ca\.crt}' | base64 -d)

kubectl apply -f - <<EOF
apiVersion: openchoreo.dev/v1alpha1
kind: ClusterDataPlane
metadata:
  name: default
spec:
  planeID: default
  clusterAgent:
    clientCA:
      value: |
$(echo "$AGENT_CA" | sed 's/^/        /')
  secretStoreRef:
    name: default
  gateway:
    ingress:
      external:
        http:
          host: openchoreoapis.localhost
          listenerName: http
          port: 19080
        name: gateway-default
        namespace: openchoreo-data-plane
EOF
```

> **Why the `sed`?** The agent CA is a multi-line PEM. The `sed 's/^/        /'` indents every line
> by 8 spaces so it nests correctly under the YAML `value: |` block. Run the whole block as one
> paste.

## Verify this step

```bash
# Data-plane pods (gateway, cluster agent, etc.) are Running
kubectl get pods -n openchoreo-data-plane

# The control plane sees the data plane and marks it registered/ready
kubectl get clusterdataplane default
kubectl get clusterdataplane default -o jsonpath='{.status.conditions}' | jq . 2>/dev/null || \
  kubectl describe clusterdataplane default | sed -n '/Conditions/,/Events/p'
```

`kubectl get clusterdataplane default` should eventually show a healthy/registered status. If it
stays not-ready, the cluster agent can't reach the gateway — re-check the CA configmap from 4.1 and
that `cluster-agent-tls` exists in `openchoreo-data-plane`.

---

Next: [Step 5 — Workflow plane »](05-workflow-plane.md)
