# network-ops-service

The OSS half of the telco chat demo: data usage records and service-disruption
reports. Plain Go (`net/http`), Postgres via `pgx`.

## Environment variables

| Variable       | Default | Description                                                                 |
|----------------|---------|-------------------------------------------------------------------------------|
| `PORT`         | `8080`  | HTTP listen port.                                                             |
| `DATABASE_URL` | *(required)* | Postgres DSN, e.g. `postgres://user:pass@host:5432/db?sslmode=disable`. |

On startup the service runs an idempotent `CREATE TABLE IF NOT EXISTS` migration for its
own tables (`usage_records`, `service_reports`) and seeds demo data only if a table is
empty, so it's safe to restart against the same database. Usage records are seeded for
the last 7 days (relative to the moment the service starts) for customer ids
`cust-001`..`cust-004` (the same ids used by `subscription-service`; this service does not
store or need customer names).

`employee-console-ui`'s Incidents tab calls this service directly from the browser (chat-gateway
is chat-only, not a data proxy — see the top-level demo README), so every response carries
wide-open CORS headers (`Access-Control-Allow-Origin: *`, no credentials) and `OPTIONS` preflight
requests get a `204` — see `cors.go`.

## Run locally

```sh
# start a local Postgres (example) — can be the same instance subscription-service uses,
# since the two services own disjoint tables
docker run --rm -d --name telco-db -e POSTGRES_USER=telco -e POSTGRES_PASSWORD=telco \
  -e POSTGRES_DB=telco -p 5432:5432 postgres:16-alpine

export DATABASE_URL="postgres://telco:telco@localhost:5432/telco?sslmode=disable"
export PORT=8081
go run .
```

## Endpoints

### Health

```sh
curl -s localhost:8081/healthz
```

### Usage

```sh
# usage for a specific date (404 if there's no record for that exact date)
curl -s 'localhost:8081/customers/cust-001/usage?date=2026-08-30'

# most recent N days, oldest first (default 7, capped at 30) — for a
# dashboard table/sparkline without looping single-date lookups
curl -s 'localhost:8081/customers/cust-001/usage/history?days=7'
```

### Service reports

```sh
# file a new report
curl -s -X POST localhost:8081/reports \
  -H 'Content-Type: application/json' \
  -d '{"customerId":"cust-002","category":"connectivity","description":"No signal indoors"}'

# list all reports
curl -s localhost:8081/reports

# filter by customer, status, and/or category (all combinable); excludeId
# drops one report by id from the results — used for a "related incidents"
# panel that shouldn't list the incident it's related to
curl -s 'localhost:8081/reports?customerId=cust-001&status=open'
curl -s 'localhost:8081/reports?category=connectivity&excludeId=rep-abcd1234'

# get a single report
curl -s localhost:8081/reports/rep-abcd1234

# update status and/or resolution notes
curl -s -X PATCH localhost:8081/reports/rep-abcd1234 \
  -H 'Content-Type: application/json' \
  -d '{"status":"in_progress"}'
```

Every request may optionally carry `X-Actor-Role` and `X-Actor-Id` headers; they are
included in the structured request log but have no effect on authorization.

## Docker

```sh
docker build -t telco-chat-demo/network-ops-service:latest .
```
