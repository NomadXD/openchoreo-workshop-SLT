# Step 0 — Prerequisites

Install these CLI tools **before** the workshop. Pulling images and charts on the day over shared
Wi-Fi is slow — do it in advance if you can.

## Required tools

| Tool | Minimum version | Check |
|------|-----------------|-------|
| Docker | v26+ (8 GB RAM, 4 CPU available) | `docker --version` |
| k3d | v5.8+ | `k3d --version` |
| kubectl | v1.32+ | `kubectl version --client` |
| Helm | v3.12+ | `helm version --short` |

Quick verification:

```bash
docker --version
k3d --version
kubectl version --client
helm version --short
docker info > /dev/null && echo "docker daemon OK"
```

## Resource requirements

The full four-plane install is heavy. Make sure your Docker/VM has at least:

- **8 GB RAM** (12 GB comfortable — the observability plane adds OpenObserve, Prometheus, and a
  Fluent Bit DaemonSet)
- **4 CPUs**
- **~30 GB free disk** for images

## macOS / Apple Silicon notes

Docker Desktop works, but many OpenChoreo contributors use **[Colima](https://github.com/abiosoft/colima)**
with the VZ backend and Rosetta:

```bash
colima start --cpu 4 --memory 12 --vm-type vz --vz-rosetta
```

> **Colima + k3d DNS gotcha:** when creating the k3d cluster on Colima you must set
> `K3D_FIX_DNS=0`, otherwise the cluster comes up with broken DNS and the `*.localhost`
> hostnames won't resolve. This is already baked into the cluster-create command in
> [step 1](01-create-cluster.md). If you later restart Colima, the in-cluster DNS to
> `host.k3d.internal` can break again and you'll see `ENOTFOUND host.k3d.internal` — recreate the
> CoreDNS rewrite (see [step 2](02-platform-prerequisites.md)) or restart the cluster.

## Windows / Linux

- **Linux:** native Docker works well; no special flags needed.
- **Windows:** use WSL2 with Docker Desktop's WSL2 backend, and run all commands inside the WSL2
  shell.

## Set your shell variables

Open the terminal you'll use for the whole workshop and export the pinned versions once:

```bash
export OPENCHOREO_REF=release-v1.2.0-rc.1
export OPENCHOREO_VERSION=1.2.0-rc.1
export RAW_BASE="https://raw.githubusercontent.com/openchoreo/openchoreo/${OPENCHOREO_REF}"
```

> If you open a new terminal later, re-export these three variables before running commands.

---

Next: [Step 1 — Create the cluster »](01-create-cluster.md)
