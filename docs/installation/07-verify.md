# Step 7 — Verify the installation

With all four planes installed, let's confirm the platform works end to end: log in to the console,
deploy a pre-built app, and (optionally) build one from source.

## 7.1 Log in to the OpenChoreo console

Open the developer console:

- **URL:** <http://openchoreo.localhost:8080>
- **Username:** `admin@openchoreo.dev`
- **Password:** `Admin@123`

If the page doesn't load, the `*.localhost` CoreDNS rewrite from
[step 2](02-platform-prerequisites.md) isn't in place, or the control-plane pods aren't ready yet.

## 7.2 Deploy a pre-built app (from image)

This deploys a React starter web app directly from a published image — no build required:

```bash
kubectl apply -f "${RAW_BASE}/samples/from-image/react-starter-web-app/react-starter.yaml"

kubectl wait --for=condition=available deployment \
  -l openchoreo.dev/component=react-starter -A --timeout=120s

HOSTNAME=$(kubectl get httproute -A -l openchoreo.dev/component=react-starter \
  -o jsonpath='{.items[0].spec.hostnames[0]}')
echo "Open: http://${HOSTNAME}:19080"
```

Open the printed URL in your browser — you should see the React starter page.

## 7.3 Build from source (optional)

Requires the [workflow plane](05-workflow-plane.md). This builds a Go service from source in-cluster,
publishes the image to the in-cluster registry, and deploys it:

```bash
kubectl apply -f "${RAW_BASE}/samples/from-source/services/go-docker-greeter/greeting-service.yaml"

# Watch the build workflow run
kubectl get workflow -n workflows-default --watch
```

Once the workflow completes, wait for the deployment and call the service:

```bash
kubectl wait --for=condition=available deployment \
  -l openchoreo.dev/component=greeting-service -A --timeout=300s

HOSTNAME=$(kubectl get httproute -A -l openchoreo.dev/component=greeting-service \
  -o jsonpath='{.items[0].spec.hostnames[0]}')
PATH_PREFIX=$(kubectl get httproute -A -l openchoreo.dev/component=greeting-service \
  -o jsonpath='{.items[0].spec.rules[0].matches[0].path.value}')

curl "http://${HOSTNAME}:19080${PATH_PREFIX}/greeter/greet"
```

You should get a greeting response.

## 7.4 See logs in OpenObserve

With a component running, its container logs flow into OpenObserve (via Fluent Bit). Open the
OpenObserve console as described in
[step 6.9 / "Open the OpenObserve console"](06-observability-plane-openobserve.md#open-the-openobserve-console),
go to **Logs**, pick the log stream, and filter for your component (e.g. `react-starter` or
`greeting-service`). Traces for instrumented services appear under **Traces**.

> Reminder ([caveats](06-observability-plane-openobserve.md#caveats-on-v11)): on v1.1 the
> OpenObserve console is the reliable place to view logs/traces. The OpenChoreo console's Runtime
> Logs tab is wired for OpenSearch and may not surface OpenObserve data.

## Full-install sanity check

A quick sweep across all planes:

```bash
for ns in openchoreo-control-plane openchoreo-data-plane \
          openchoreo-workflow-plane openchoreo-observability-plane; do
  echo "=== $ns ==="
  kubectl get pods -n "$ns" --no-headers | awk '{print $3}' | sort | uniq -c
done

echo "=== registered planes ==="
kubectl get clusterdataplane,clusterworkflowplane,clusterobservabilityplane
```

Every plane's pods should be `Running` (a few `Completed` job pods are fine), and all three
`Cluster*Plane` resources should exist.

---

Next: [Step 8 — Cleanup »](08-cleanup.md)
