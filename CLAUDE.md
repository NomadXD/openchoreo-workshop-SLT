# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A **documentation-only workshop** for OpenChoreo (an open-source Internal Developer Platform for
Kubernetes). There is no application code here — the deliverable is the installation guide under
`docs/installation/`, which walks a reader through standing up a full OpenChoreo environment on a
local [k3d](https://k3d.io) cluster.

> Do not confuse this repo with the OpenChoreo **source** repo. Parent-directory `CLAUDE.md` files
> (`~/Dev/CLAUDE.md`, `~/Dev/Projects/CLAUDE.md`) describe the Go/Kubebuilder source project at
> `github.com/openchoreo/openchoreo`. That guidance (build/lint/test/make targets) does **not**
> apply here — this repo has no Go code, no Makefile, and no test suite.

## Structure

```
README.md                       # Landing page: audience, version matrix, step index
docs/installation/
  README.md                     # Installation overview: the four planes + version table
  00-prerequisites.md .. 08-cleanup.md   # Ordered steps, read/executed in sequence
```

Untracked working files may appear at the repo root (e.g. a downloaded Helm chart directory like
`openchoreo-control-plane/`, or scratch YAML). These are local scratch used while verifying the
guide; `.gitignore` covers `*.tgz`, `scratch/`, `kubeconfig*`, etc. Don't commit them.

## The installation guide

The guide installs OpenChoreo's **four planes** into one k3d cluster: **control**, **data**,
**workflow**, and **observability** (this workshop uses **OpenObserve** for logs/traces instead of
the default OpenSearch). Steps are numbered and strictly ordered — later planes register with the
control plane and depend on earlier steps' resources (e.g. the `cluster-gateway-ca` certificate).

### Conventions to preserve when editing

- **Version pinning is the whole point.** Every install command is pinned to an exact chart
  version. Versions live in three places that must stay in sync:
  1. `README.md` "Version matrix" table
  2. `docs/installation/README.md` "Versions" table (the `#versions` anchor) — the canonical table
  3. Inline `--version` flags and `helm`/`kubectl` commands in each step
- **Shell variables drive the commands.** Steps assume these are exported (defined in
  `00-prerequisites.md` and the overview):
  ```bash
  export OPENCHOREO_REF=release-v1.2.0-rc.1  # git ref/branch for raw manifests
  export OPENCHOREO_VERSION=1.2.0-rc.1       # chart version passed to --version
  export RAW_BASE="https://raw.githubusercontent.com/openchoreo/openchoreo/${OPENCHOREO_REF}"
  ```
  Commands fetch manifests from `${RAW_BASE}/...` and install charts from
  `oci://ghcr.io/openchoreo/helm-charts/*` at `${OPENCHOREO_VERSION}`. **A version bump means
  updating both `OPENCHOREO_REF` and `OPENCHOREO_VERSION` everywhere, plus the tables.**
- **Namespaces are fixed:** `openchoreo-control-plane`, `openchoreo-data-plane`,
  `openchoreo-workflow-plane`, `openchoreo-observability-plane`.
- **Each step ends with a "Verify this step" block** (a `kubectl wait` / `kubectl get`) and a
  `Next: [Step N …»]` footer link. Keep both when adding or reordering steps.
- **The OpenObserve swap is the workshop's main divergence from the stock install.** The stock v1.2
  install uses OpenSearch for logs/traces; this workshop installs the OpenObserve modules instead
  and enables the observer's `logsAdapter`/`tracingAdapter` so OpenObserve data shows in the
  OpenChoreo console. That wiring, and the OpenObserve module versions, changed meaningfully from
  the v1.1 line — re-verify the observer adapter config and module compatibility (community-modules
  READMEs) whenever the pinned OpenChoreo version changes, not just the version numbers.

## Verifying doc changes

There is no build or test. To validate a change to the guide, the guide's own commands are the
test: run them against a fresh k3d cluster in sequence and confirm each step's "Verify this step"
block passes before moving on. When bumping versions, confirm the referenced chart version and the
`${OPENCHOREO_REF}` raw-manifest paths actually exist upstream before updating the docs.

## References

- OpenChoreo docs — <https://openchoreo.dev/docs/>
- Try it on k3d locally — <https://openchoreo.dev/docs/getting-started/try-it-out/on-k3d-locally/>
- OpenChoreo GitHub — <https://github.com/openchoreo/openchoreo>
- Community modules — <https://github.com/openchoreo/community-modules>
