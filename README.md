RA OpenLAN Network Topology Service
===================================

This service exposes a TLS-protected HTTP API that builds the current wireless topology for a given board. It:
- Consumes lifecycle events from Kafka to keep an in-memory registry of reachable openwifi services (`internal/store`, `internal/services/discovery`).
- Calls discovered services (e.g., timepoints/owanalytics, owsec) through a shared OpenAPI client (`internal/adapters/serviceclient`) with API-key headers and optional custom CA.
- Builds a topology graph from timepoint samples: normalizes BSSIDs, separates AP vs mesh faces, attaches clients, and derives mesh edges (`internal/services/topology_service.go`).
- Publishes its own lifecycle events back to Kafka for discovery by other services (`internal/services/lifecycle` via the Kafka producer).
- Serves `/v1/topology` behind API-key or bearer-token validation (`internal/http`, `internal/http/middlewares`, `internal/security`), plus `/livez` and `/readyz` health probes.
- Initializes a Postgres pool (reserved for timepoint DB access) and structured logging (`internal/logger`).

HTTP API
--------
- `GET /v1/topology?boardId={id}&fromDate={RFC3339}&endDate={RFC3339}&maxRecords={n}` (see `internal/models.TimepointsQuery` for all optional params)
  - Auth: `X-API-KEY: <API_KEY>` for internal calls; otherwise bearer token validated via owsec.
  - Response: `models.Topology` JSON with nodes (devices + faces), mesh edges, and timestamp. Timestamps are emitted in Asia/Kolkata for faces; overall timestamp is UTC.

Configuration (env vars)
------------------------
Key settings parsed in `internal/config/config.go` (defaults in code):
- App/HTTP: `APP_NAME`, `APP_ENV`, `HTTP_PORT` (default 8088), `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`.
- Auth/TLS: `API_KEY`, `INTERNAL_RESTAPI_HOST_ROOTCA` (CA for token validation client), `INTERNAL_RESTAPI_HOST_CERT`/`INTERNAL_RESTAPI_HOST_KEY` (TLS cert/key required to start).
- Postgres: `STORAGE_TYPE_POSTGRESQL_HOST|PORT|USERNAME|PASSWORD|DATABASE|SSLMODE|MAX_CONNS|MIN_CONNS|MAX_CONN_LIFETIME`.
- Kafka: `KAFKA_BROKERS` (comma separated), `KAFKA_GROUP_ID`, `KAFKA_TOPIC_CMD`, `KAFKA_TOPIC_RESP`, `KAFKA_TOPIC_LIFECYCLE`, `KAFKA_DIAL_TIMEOUT`, `KAFKA_WRITE_TIMEOUT`, `KAFKA_READ_TIMEOUT`, `KAFKA_MIN_BYTES`, `KAFKA_MAX_BYTES`, `KAFKA_ALLOW_AUTO_CREATE`.
- Topology/requests: `TOPOLOGY_WINDOW`, `TOPOLOGY_DRIFT`, `REQUEST_TIMEOUT`.
- Lifecycle identity: `SERVICE_TYPE` (default `nwtopology`), `LIFECYCLE_INTERVAL`, `PRIVATE_ENDPOINT`, `PUBLIC_ENDPOINT`, `BUILD_VERSION`.
- Logging: `SYSTEM_LOG_LEVEL`, `SYSTEM__LOG_JSON`, `SYSTEM__LOG_FILE`, rotation settings.

Sample local env (adapt paths/brokers as needed):
```
export API_KEY=dev-secret-key
export HTTP_PORT=8088
export KAFKA_BROKERS=localhost:9092
export KAFKA_TOPIC_LIFECYCLE=service_events
export KAFKA_GROUP_ID=nwtopology-service-group
export PRIVATE_ENDPOINT=https://127.0.0.1:8088
export PUBLIC_ENDPOINT=https://127.0.0.1:8088
export INTERNAL_RESTAPI_HOST_CERT=./certs/restapi-cert.pem
export INTERNAL_RESTAPI_HOST_KEY=./certs/restapi-key.pem
export INTERNAL_RESTAPI_HOST_ROOTCA=./certs/restapi-ca.pem
```

Build & Run Locally (Go 1.25+)
------------------------------
1) Install deps: `go mod tidy`.
2) Build: `go build -o bin/nw-topology ./cmd/main.go`.
3) Run with env set (TLS cert/key are mandatory): `./bin/nw-topology` or `go run ./cmd/main.go`.
4) Test the API (replace key/board):  
   `curl -k -H "X-API-KEY: $API_KEY" "https://localhost:8088/v1/topology?boardId=<BOARD_ID>&maxRecords=50"`.
5) Validate: `go test ./...` (also runs during Docker build).

Container Builds
----------------
- Docker: `docker build -t network-topology .` then
  `docker-compose up`.
- Compose (uses `settings.local.env` and mounted certs): `docker-compose up --build`.

Code Map (quick pointers)
-------------------------
- `cmd/main.go`: wiring/bootstrap, TLS HTTP server, Kafka producers/consumer, lifecycle service start.
- `internal/http`: Fiber server setup, auth middleware, route registration, topology handler.
- `internal/services`: topology builder logic, lifecycle publisher, publisher interface.
- `adapters/kafka`: Kafka consumer/producer implementations; handler registry lives in `internal/kafka`.
- `internal/adapters/serviceclient`: HTTP client for owanalytics/owsec; timepoints fetcher.
- `internal/store`: in-memory discovery store fed by Kafka lifecycle events (`internal/services/discovery`).
- `internal/repositories`: Postgres access scaffolding for timepoints.
