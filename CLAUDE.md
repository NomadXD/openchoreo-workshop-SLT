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
README.md                       # Landing page: audience, version matrix, step index, scenarios
docs/installation/
  README.md                     # Installation overview: the four planes + version table
  00-prerequisites.md .. 08-cleanup.md   # Ordered steps, read/executed in sequence
docs/scenarios/
  README.md                     # Scenarios index (table of scenarios)
  01-build-from-source/         # Build the Go greeter from Git (workflow plane)
  02-doclet/                    # Multi-service app + self-service Postgres/NATS resources
    README.md                   # Portal-driven walkthrough, ordered steps
    images/                     # Screenshots referenced by the walkthrough
```

Prose is hard-wrapped at ~100 columns; keep that when editing.

Untracked working files may appear at the repo root (e.g. a downloaded Helm chart directory like
`openchoreo-control-plane/`, or scratch YAML such as `projects-crd-relaxed.yaml`). These are local
scratch used while verifying the guide. **`.gitignore` does not cover all of them** (it covers
`*.tgz`, `scratch/`, `kubeconfig*`, `/tmp/`, editor dirs), so never `git add -A`/`git commit -a` —
stage the doc paths explicitly.

## Scenarios

Scenarios run **on top of** the installed environment and are **console-driven** (Backstage
developer portal at `http://openchoreo.localhost:8080`), not `kubectl`-driven — the opposite
emphasis from the installation guide. `kubectl` appears only in their "Verify this step" blocks.

- **Scenario 1** deploys the Go greeter (`openchoreo/sample-workloads` → `service-go-greeter`) via
  **Build from Source** with the `dockerfile-builder` workflow.
- **Scenario 2** (Doclet) deploys three prebuilt images (`ghcr.io/openchoreo/samples/doclet-*`)
  plus **Resource** instances (Postgres, NATS), then wires them via resource-output → env-var
  bindings and component→endpoint dependencies. Everything lands in namespace `default` / project
  `default`, Development only, built bottom-up (resources → services → frontend) because the
  dependency pickers only see what already exists.

Each scenario README follows the same anatomy — keep it when adding one: title with time estimate,
a `mermaid` diagram of the topology, **Before you start** (console URL + credentials), a
**What we'll build** table of exact field values, numbered console steps each embedding a
screenshot from `images/`, **Verify this step** blocks, **What you did**, an optional per-scenario
**Clean up** (`kubectl delete component.openchoreo.dev …`), and a `Next:` footer link.

Screenshots are the source of truth for the UI flow: when a flow changes, drive the live console
with the browser tools and re-capture into `images/` (numbered `NN-name.png`) rather than
hand-editing prose alone.

**Adding a scenario means editing three files:** the new `docs/scenarios/NN-slug/README.md`, the
scenarios index table (`docs/scenarios/README.md`), and the Scenarios table in the root
`README.md`. Both tables carry the same columns (#, link, what you learn, time) — keep them in sync.

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
- **Fixed environment facts** the docs repeat — change them in one place and every reference must
  follow:
  | Thing | Value |
  |---|---|
  | k3d cluster / kubectl context / node | `openchoreo` / `k3d-openchoreo` / `k3d-openchoreo-server-0` |
  | OpenChoreo console | <http://openchoreo.localhost:8080> — `admin@openchoreo.dev` / `Admin@123` |
  | Deployed component endpoints | `http://<httproute hostname>:19080<pathPrefix>` |
  | OpenObserve console | port-forward `5080` — `admin@openchoreo.localhost` / `ThisIsTheOpenObservePassword1` |
  | Observer URL registered in `ClusterObservabilityPlane` | `http://observer.openchoreo.localhost:11080` |
  | OpenBao (dev mode, secret backend) | pod `openbao-0` in ns `openbao`, root token `root`, `127.0.0.1:8200` |
  Credentials are intentionally published workshop defaults, not secrets.
- **Cluster config comes from upstream, not this repo:** the cluster is created by piping
  `${RAW_BASE}/install/k3d/single-cluster/config.yaml` into `k3d cluster create`, and planes are
  installed with `--values ${RAW_BASE}/install/k3d/single-cluster/values-*.yaml`. There are no local
  values files to edit — divergences from stock are expressed as `--set` flags or extra manifests in
  the step itself.
- **Colima / Apple Silicon carve-outs** (`K3D_FIX_DNS=0` on cluster create, the
  `host.k3d.internal` / CoreDNS-rewrite breakage after a Colima restart) are called out in steps 0
  and 1. Keep those notes — most workshop attendees are on macOS.
- **Each step ends with a "Verify this step" block** (a `kubectl wait` / `kubectl get`) and a
  `Next: [Step N …»]` footer link. Keep both when adding or reordering steps.
- **The OpenObserve swap is the workshop's main divergence from the stock install.** The stock v1.2
  install uses OpenSearch for logs/traces; this workshop installs the OpenObserve modules instead,
  seeds their OpenBao credentials by hand, and relies on the observer's default
  `logsAdapter`/`tracingAdapter` URLs (`logs-adapter:9098` / `tracing-adapter:9100`) so OpenObserve
  data shows in the OpenChoreo console. That wiring, and the OpenObserve module versions, changed
  meaningfully from the v1.1 line — re-verify the observer adapter config and module compatibility
  (community-modules READMEs) whenever the pinned OpenChoreo version changes, not just the version
  numbers.

## Verifying doc changes

There is no build or test. To validate a change to the guide, the guide's own commands are the
test: run them against a fresh k3d cluster in sequence and confirm each step's "Verify this step"
block passes before moving on. When bumping versions, confirm the referenced chart version and the
`${OPENCHOREO_REF}` raw-manifest paths actually exist upstream before updating the docs — e.g.:

```bash
helm show chart oci://ghcr.io/openchoreo/helm-charts/openchoreo-control-plane --version "$OPENCHOREO_VERSION"
curl -fsI "${RAW_BASE}/install/k3d/single-cluster/config.yaml"
```

Chart values also drift between releases in ways version bumps alone won't reveal — a key the docs
`--set` may vanish from the schema and hard-fail the install (see the `logsAdapter.enabled` note in
step 6.5). Check with `helm show values <chart> --version <v>` before trusting an existing `--set`.

A partial re-verify is often enough: steps 3–6 are independent installs on top of steps 1–2, so a
change to one plane can be tested by reinstalling just that plane's chart into an existing cluster.
Full teardown is `k3d cluster delete openchoreo` (step 8).

## References

- OpenChoreo docs — <https://openchoreo.dev/docs/>
- Try it on k3d locally — <https://openchoreo.dev/docs/getting-started/try-it-out/on-k3d-locally/>
- OpenChoreo GitHub — <https://github.com/openchoreo/openchoreo>
- Community modules — <https://github.com/openchoreo/community-modules>
