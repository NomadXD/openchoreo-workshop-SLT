# Step 1 — Create the k3d cluster

We create a single k3d cluster named **`openchoreo`**. Its port mappings and node layout come from
the OpenChoreo project's tested `single-cluster` config, so the `*.localhost` hostnames and gateway
ports line up with the rest of the guide.

## Create the cluster

```bash
curl -fsSL "${RAW_BASE}/install/k3d/single-cluster/config.yaml" | k3d cluster create --config=-
```

This creates a cluster named `openchoreo` and sets your kubectl context to **`k3d-openchoreo`**.

> **On Colima / Apple Silicon**, prefix the command with `K3D_FIX_DNS=0` so DNS is set up correctly:
>
> ```bash
> curl -fsSL "${RAW_BASE}/install/k3d/single-cluster/config.yaml" \
>   | K3D_FIX_DNS=0 k3d cluster create --config=-
> ```

## Point kubectl at the cluster

k3d switches context automatically, but confirm it:

```bash
kubectl config use-context k3d-openchoreo
kubectl cluster-info
```

## Verify this step

```bash
# Cluster is listed and running
k3d cluster list

# Nodes are Ready (1 server + 2 agents)
kubectl get nodes
```

You should see three nodes in `Ready` state:

```
NAME                       STATUS   ROLES                  AGE   VERSION
k3d-openchoreo-server-0    Ready    control-plane,master   1m    v1.32.x+k3s1
k3d-openchoreo-agent-0     Ready    <none>                 1m    v1.32.x+k3s1
k3d-openchoreo-agent-1     Ready    <none>                 1m    v1.32.x+k3s1
```

> **Troubleshooting**
> - *Port already allocated* — another k3d cluster (or process) is holding the ports. Delete the old
>   cluster with `k3d cluster delete openchoreo` or free the port, then retry.
> - *Nodes stuck `NotReady`* — give it another minute; if it persists, check `docker info` for
>   enough memory/CPU.

---

Next: [Step 2 — Platform prerequisites »](02-platform-prerequisites.md)
