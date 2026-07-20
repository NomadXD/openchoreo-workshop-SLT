# OpenChoreo Workshop — SLT

Hands-on training materials for **OpenChoreo**, an open-source Internal Developer Platform (IDP)
for Kubernetes.

This repository starts with a complete **installation guide** that stands up a full OpenChoreo
environment on a local [k3d](https://k3d.io) cluster:

- **Control plane** — the OpenChoreo API, controllers, and the Backstage-based developer console
- **Data plane** — where your components actually run
- **Workflow plane** — build-from-source pipelines (Argo Workflows + an in-cluster registry)
- **Observability plane** — logs, traces, and metrics — using the **OpenObserve** logs and
  tracing modules instead of the default OpenSearch modules

Later sections (added over time) will layer on components and end-to-end scenarios you can deploy
onto this environment.

---

## Who this is for

Developers and platform engineers who want to try OpenChoreo end to end on their own laptop.
No prior OpenChoreo experience is assumed. Comfort with `kubectl`, `helm`, and a terminal is
expected.

## What you need

A machine that can run Docker with **8 GB RAM / 4 CPU** free, plus the CLI tools listed in
[docs/installation/00-prerequisites.md](docs/installation/00-prerequisites.md). Budget **~45–60
minutes** for the full install (most of it is images pulling).

---

## Start here

Work through the installation guide in order:

| # | Step | Guide |
|---|------|-------|
| 0 | Install the required CLI tools | [00-prerequisites.md](docs/installation/00-prerequisites.md) |
| 1 | Create the k3d cluster | [01-create-cluster.md](docs/installation/01-create-cluster.md) |
| 2 | Install platform prerequisites | [02-platform-prerequisites.md](docs/installation/02-platform-prerequisites.md) |
| 3 | Install the control plane | [03-control-plane.md](docs/installation/03-control-plane.md) |
| 4 | Install the data plane | [04-data-plane.md](docs/installation/04-data-plane.md) |
| 5 | Install the workflow plane | [05-workflow-plane.md](docs/installation/05-workflow-plane.md) |
| 6 | Install the observability plane (OpenObserve) | [06-observability-plane-openobserve.md](docs/installation/06-observability-plane-openobserve.md) |
| 7 | Verify the installation | [07-verify.md](docs/installation/07-verify.md) |
| 8 | Tear everything down | [08-cleanup.md](docs/installation/08-cleanup.md) |

Or jump to the [installation overview](docs/installation/README.md) for the architecture and a
one-page summary.

---

## Scenarios

Once the environment is up, work through the hands-on scenarios — portal-driven walkthroughs (with
screenshots) that deploy real workloads onto your cluster. See the
[scenarios index](docs/scenarios/README.md).

| # | Scenario | What you learn | Time |
|---|----------|----------------|------|
| 1 | [Build a service from source](docs/scenarios/01-build-from-source/README.md) | Deploy the Go greeter straight from Git — OpenChoreo builds the image in-cluster from a `Dockerfile`, then deploys and exposes it | ~15 min |

---

## Version matrix

This workshop is pinned to a specific, tested set of versions. See the
[installation overview](docs/installation/README.md#versions) for the full table.

| Component | Version |
|-----------|---------|
| OpenChoreo | `v1.2.0-rc.1` (release candidate) |
| OpenObserve logs module | `0.5.1` |
| OpenObserve tracing module | `0.2.4` |
| Metrics (Prometheus) module | `0.6.1` |

> **Note on OpenObserve + v1.2:** OpenObserve is a first-class, documented backend on the OpenChoreo
> **v1.2** line. The stock v1.2 install uses OpenSearch for logs and traces; this workshop swaps in
> **OpenObserve** instead. Because the v1.2 *observer* service talks to pluggable logs/tracing
> adapters (pointed at the OpenObserve modules by default via `observer.logsAdapter.url` /
> `observer.tracingAdapter.url`), OpenObserve-backed logs and traces now surface in the
> **OpenChoreo console** as well as the OpenObserve console. The setup is described inline in
> [step 6](docs/installation/06-observability-plane-openobserve.md).

## References

- OpenChoreo docs — <https://openchoreo.dev/docs/>
- Try it on k3d locally — <https://openchoreo.dev/docs/getting-started/try-it-out/on-k3d-locally/>
- OpenObserve logs module — <https://openchoreo.dev/ecosystem/item/?id=logs-openobserve>
- OpenObserve tracing module — <https://openchoreo.dev/ecosystem/item/?id=tracing-openobserve>
- OpenChoreo GitHub — <https://github.com/openchoreo/openchoreo>
- Community modules — <https://github.com/openchoreo/community-modules>
