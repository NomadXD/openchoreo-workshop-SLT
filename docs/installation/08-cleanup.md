# Step 8 — Cleanup

## Delete the whole cluster (recommended)

Everything in this workshop lives inside the single k3d cluster, so the simplest, most complete
cleanup is to delete it:

```bash
k3d cluster delete openchoreo
```

This removes all planes, workloads, secrets, and the registry. Nothing persists on your host except
the images cached in Docker.

## Reclaim disk (optional)

The workshop pulls a lot of images. To reclaim that space:

```bash
docker image prune -a
```

> `docker image prune -a` removes **all** images not used by a running container, not just
> OpenChoreo's. Skip it if you have other images you want to keep.

## Uninstall individual planes (optional)

If you want to keep the cluster but remove just one plane, uninstall its Helm releases and delete its
namespace. For example, the observability plane:

```bash
helm uninstall observability-tracing-openobserve -n openchoreo-observability-plane
helm uninstall observability-logs-openobserve    -n openchoreo-observability-plane
helm uninstall observability-metrics-prometheus  -n openchoreo-observability-plane
helm uninstall openchoreo-observability-plane    -n openchoreo-observability-plane
kubectl delete clusterobservabilityplane default
kubectl delete namespace openchoreo-observability-plane
```

The same pattern applies to the data and workflow planes (uninstall the release, delete the
`Cluster*Plane` resource, delete the namespace).

---

Back to the [installation overview](README.md) · [workshop home](../../README.md)
