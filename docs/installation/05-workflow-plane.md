# Step 5 — Workflow plane

The workflow plane runs **build-from-source** pipelines (Argo Workflows) and hosts an in-cluster
container registry for the images those builds produce. It follows the same pattern as the data
plane: trust the CA, install the plane, install the workflow templates, register it.

> If you only care about deploying **pre-built images**, the workflow plane is optional. Install it
> if you want to try the build-from-source scenario in [step 7](07-verify.md#build-from-source-optional).

## 5.1 Create the namespace and trust the control-plane CA

```bash
kubectl create namespace openchoreo-workflow-plane --dry-run=client -o yaml | kubectl apply -f -

CA_CRT=$(kubectl get secret cluster-gateway-ca \
  -n openchoreo-control-plane -o jsonpath='{.data.ca\.crt}' | base64 -d)

kubectl create configmap cluster-gateway-ca \
  --from-literal=ca.crt="$CA_CRT" \
  -n openchoreo-workflow-plane --dry-run=client -o yaml | kubectl apply -f -
```

## 5.2 Install the in-cluster container registry

```bash
helm repo add twuni https://twuni.github.io/docker-registry.helm
helm repo update

helm upgrade --install registry twuni/docker-registry \
  --namespace openchoreo-workflow-plane --create-namespace \
  --values "${RAW_BASE}/install/k3d/single-cluster/values-registry.yaml"
```

## 5.3 Install the workflow plane

```bash
helm upgrade --install openchoreo-workflow-plane oci://ghcr.io/openchoreo/helm-charts/openchoreo-workflow-plane \
  --version "${OPENCHOREO_VERSION}" \
  --namespace openchoreo-workflow-plane \
  --values "${RAW_BASE}/install/k3d/single-cluster/values-wp.yaml"
```

## 5.4 Install the workflow templates

These define the standard build steps (checkout, build image, publish, generate workload):

```bash
kubectl apply \
  -f "${RAW_BASE}/samples/getting-started/workflow-templates/checkout-source.yaml" \
  -f "${RAW_BASE}/samples/getting-started/workflow-templates.yaml" \
  -f "${RAW_BASE}/samples/getting-started/workflow-templates/publish-image-k3d.yaml" \
  -f "${RAW_BASE}/samples/getting-started/workflow-templates/generate-workload-k3d.yaml"
```

## 5.5 Register the workflow plane with the control plane

```bash
kubectl wait -n openchoreo-workflow-plane \
  --for=jsonpath='{.data.ca\.crt}' secret/cluster-agent-tls --timeout=120s

AGENT_CA=$(kubectl get secret cluster-agent-tls \
  -n openchoreo-workflow-plane -o jsonpath='{.data.ca\.crt}' | base64 -d)

kubectl apply -f - <<EOF
apiVersion: openchoreo.dev/v1alpha1
kind: ClusterWorkflowPlane
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
EOF
```

## Verify this step

```bash
# Registry + workflow-plane pods are Running
kubectl get pods -n openchoreo-workflow-plane

# Argo Workflows CRDs and templates are present
kubectl get clusterworkflowtemplate -A 2>/dev/null || kubectl get workflowtemplate -A

# Control plane sees the workflow plane
kubectl get clusterworkflowplane default
```

---

Next: [Step 6 — Observability plane (OpenObserve) »](06-observability-plane-openobserve.md)
