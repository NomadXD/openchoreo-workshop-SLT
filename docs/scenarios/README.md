# Scenarios

Hands-on scenarios you run **on top of** the environment from the
[installation guide](../installation/README.md). Each one is a self-contained, portal-driven
walkthrough — create something in the OpenChoreo console, deploy it, and reach it — with screenshots
for every step.

> **Prerequisite:** a running OpenChoreo environment (all four planes). If you haven't installed it
> yet, start with the [installation guide](../installation/README.md). Confirm the console loads at
> <http://openchoreo.localhost:8080> before beginning any scenario.

| # | Scenario | What you learn | Time |
|---|----------|----------------|------|
| 1 | [Build a service from source](01-build-from-source/README.md) | Deploy the Go greeter straight from Git — OpenChoreo builds the image in-cluster from a `Dockerfile`, then deploys and exposes it | ~15 min |
| 2 | [A multi-service app with self-service infra](02-doclet/README.md) | Provision managed Postgres and NATS on demand, then deploy and wire together a three-component app (Doclet) — resource and component dependencies, all in the console | ~25 min |

More scenarios will be added over time (promote across environments and observability).
