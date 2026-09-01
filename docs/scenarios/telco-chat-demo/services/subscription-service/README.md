# subscription-service

The BSS half of the telco chat demo: customer accounts, the plan catalog, and
per-customer subscriptions. Plain Go (`net/http`), Postgres via `pgx`.

## Environment variables

| Variable       | Default | Description                                                                 |
|----------------|---------|-------------------------------------------------------------------------------|
| `PORT`         | `8080`  | HTTP listen port.                                                             |
| `DATABASE_URL` | *(required)* | Postgres DSN, e.g. `postgres://user:pass@host:5432/db?sslmode=disable`. |

On startup the service runs an idempotent `CREATE TABLE IF NOT EXISTS` migration for its
own tables (`customers`, `plans`, `subscriptions`) and seeds demo data only if a table is
empty, so it's safe to restart against the same database.

## Run locally

```sh
# start a local Postgres (example)
docker run --rm -d --name telco-db -e POSTGRES_USER=telco -e POSTGRES_PASSWORD=telco \
  -e POSTGRES_DB=telco -p 5432:5432 postgres:16-alpine

export DATABASE_URL="postgres://telco:telco@localhost:5432/telco?sslmode=disable"
export PORT=8080
go run .
```

## Endpoints

### Health

```sh
curl -s localhost:8080/healthz
```

### Plans

```sh
# list active plans
curl -s localhost:8080/plans

# create a plan
curl -s -X POST localhost:8080/plans \
  -H 'Content-Type: application/json' \
  -d '{"id":"plan-corp-500","name":"Corporate 500GB","dataGb":500,"priceCents":999000}'

# update a plan
curl -s -X PUT localhost:8080/plans/plan-corp-500 \
  -H 'Content-Type: application/json' \
  -d '{"name":"Corporate 500GB","dataGb":500,"priceCents":899000}'

# soft-delete a plan
curl -s -X DELETE localhost:8080/plans/plan-corp-500 -o /dev/null -w '%{http_code}\n'
```

### Customers

```sh
# list all customers
curl -s localhost:8080/customers

# search by id, name, or msisdn (case-insensitive substring)
curl -s 'localhost:8080/customers?search=perera'

# get one customer
curl -s localhost:8080/customers/cust-001
```

### Subscriptions

```sh
# get a customer's current subscription
curl -s localhost:8080/customers/cust-001/subscription

# change a customer's plan
curl -s -X POST localhost:8080/customers/cust-001/subscription \
  -H 'Content-Type: application/json' \
  -d '{"planId":"plan-unlimited"}'
```

Every request may optionally carry `X-Actor-Role` and `X-Actor-Id` headers; they are
included in the structured request log but have no effect on authorization.

## Docker

```sh
docker build -t telco-chat-demo/subscription-service:latest .
```
