# Installation Guide — Overview

This guide installs a complete OpenChoreo environment on a single local **k3d** cluster, one plane
at a time, so you understand what each piece does. By the end you will have all four planes running
and will deploy a sample application to confirm everything works.

## The four planes

OpenChoreo is organised into cooperating **planes**. In this workshop they all run inside one k3d
cluster, but in production they are typically separate clusters.

```
┌──────────────────────────────────────────────────────────────────────┐
│                          k3d cluster "openchoreo"                      │
│                                                                        │
│  ┌────────────────────┐        ┌────────────────────┐                 │
│  │   Control plane    │        │     Data plane     │                 │
│  │  API + controllers │◀──────▶│  runs your compo-  │                 │
│  │  Backstage console │        │  nents (workloads) │                 │
│  └─────────┬──────────┘        └─────────┬──────────┘                 │
│            │                             │                            │
│            ▼                             ▼                            │
│  ┌────────────────────┐        ┌────────────────────┐                 │
│  │   Workflow plane   │        │ Observability plane │                │
│  │  build-from-source │        │  logs / traces /    │                │
│  │  (Argo + registry) │        │  metrics            │                │
│  └────────────────────┘        │  (OpenObserve here) │                │
│                                 └────────────────────┘                │
└──────────────────────────────────────────────────────────────────────┘
```

| Plane | Namespace | What it does |
|-------|-----------|--------------|
| **Control** | `openchoreo-control-plane` | Hosts the OpenChoreo API server, the controllers that reconcile your resources, and the Backstage-based developer console. This is where you (and the platform) drive everything. |
| **Data** | `openchoreo-data-plane` | Where your components actually run. Ingress/egress is mediated by a gateway. |
| **Workflow** | `openchoreo-workflow-plane` | Runs build-from-source pipelines (Argo Workflows) and hosts an in-cluster container registry for the images they produce. |
| **Observability** | `openchoreo-observability-plane` | Collects logs, traces, and metrics from the other planes. In this workshop we use **OpenObserve** for logs and traces (instead of the default OpenSearch) and Prometheus for metrics. |

The planes talk to the control plane through a **cluster gateway / cluster agent** pair (mTLS), so
each plane is registered with the control plane after it is installed (`ClusterDataPlane`,
`ClusterWorkflowPlane`, `ClusterObservabilityPlane` resources).

## Supporting infrastructure

Before any plane is installed, a set of platform prerequisites is deployed
([step 2](02-platform-prerequisites.md)):

- **Gateway API CRDs** + **kgateway** — the ingress/gateway implementation
- **cert-manager** — issues the internal mTLS certificates the planes use
- **External Secrets Operator** + **OpenBao** — secret management; OpenBao is the secret backend and
  ExternalSecrets project its values into Kubernetes secrets
- **CoreDNS rewrite** — resolves the `*.localhost` hostnames used by the console and gateways

## <a id="versions"></a>Versions

Every command in this guide is pinned to these versions. If you copy commands elsewhere, keep them
consistent.

| Component | Version | Source |
|-----------|---------|--------|
| OpenChoreo (all core charts) | `1.2.0-rc.1` (git ref `release-v1.2.0-rc.1`) | `oci://ghcr.io/openchoreo/helm-charts/*` |
| OpenObserve **logs** module | `0.5.1` | `oci://ghcr.io/openchoreo/helm-charts/observability-logs-openobserve` |
| OpenObserve **tracing** module | `0.2.4` | `oci://ghcr.io/openchoreo/helm-charts/observability-tracing-openobserve` |
| **metrics** module (Prometheus) | `0.6.1` | `oci://ghcr.io/openchoreo/helm-charts/observability-metrics-prometheus` |
| Thunder (identity) | `0.28.0` | `oci://ghcr.io/asgardeo/helm-charts/thunder` |
| Gateway API CRDs | `v1.5.1` | `github.com/kubernetes-sigs/gateway-api` |
| cert-manager | `v1.19.4` | `oci://quay.io/jetstack/charts/cert-manager` |
| External Secrets Operator | `2.0.1` | `oci://ghcr.io/external-secrets/charts/external-secrets` |
| kgateway | `v2.3.1` | `oci://cr.kgateway.dev/kgateway-dev/charts/kgateway` |
| OpenBao | `0.25.6` | `oci://ghcr.io/openbao/charts/openbao` |

### Shell variables used throughout

Export these once per terminal session; the guide's commands reference them:

```bash
export OPENCHOREO_REF=release-v1.2.0-rc.1
export OPENCHOREO_VERSION=1.2.0-rc.1
export RAW_BASE="https://raw.githubusercontent.com/openchoreo/openchoreo/${OPENCHOREO_REF}"
```

## The steps

1. [Prerequisites](00-prerequisites.md) — CLI tools and resources
2. [Create the cluster](01-create-cluster.md)
3. [Platform prerequisites](02-platform-prerequisites.md)
4. [Control plane](03-control-plane.md)
5. [Data plane](04-data-plane.md)
6. [Workflow plane](05-workflow-plane.md)
7. [Observability plane (OpenObserve)](06-observability-plane-openobserve.md)
8. [Verify](07-verify.md)
9. [Cleanup](08-cleanup.md)

> **Tip:** Each step ends with a *"Verify this step"* block. Don't move on until it passes — a plane
> that isn't healthy will make later steps fail in confusing ways.
