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

## Version matrix

This workshop is pinned to a specific, tested set of versions. See the
[installation overview](docs/installation/README.md#versions) for the full table.

| Component | Version |
|-----------|---------|
| OpenChoreo | `v1.1.2` (GA) |
| OpenObserve logs module | `0.4.2` |
| OpenObserve tracing module | `0.2.4` |
| Metrics (Prometheus) module | `0.6.1` |

> **Note on OpenObserve + v1.1:** OpenObserve is a first-class, documented backend on the
> OpenChoreo **v1.2** line. On **v1.1.2** the OpenObserve modules install and collect data, but the
> control-plane *observer* service is wired for OpenSearch, so in this workshop you view logs and
> traces directly in the **OpenObserve console**. The relevant caveats are called out inline in
> [step 6](docs/installation/06-observability-plane-openobserve.md#caveats-on-v11). If you would
> rather have the console-integrated experience, use the v1.2 line instead.

## References

- OpenChoreo docs — <https://openchoreo.dev/docs/>
- Try it on k3d locally — <https://openchoreo.dev/docs/getting-started/try-it-out/on-k3d-locally/>
- OpenObserve logs module — <https://openchoreo.dev/ecosystem/item/?id=logs-openobserve>
- OpenObserve tracing module — <https://openchoreo.dev/ecosystem/item/?id=tracing-openobserve>
- OpenChoreo GitHub — <https://github.com/openchoreo/openchoreo>
- Community modules — <https://github.com/openchoreo/community-modules>
