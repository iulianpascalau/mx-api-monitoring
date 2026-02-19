# mx-api-monitoring — Full Technical Specification

## 1. Overview

The system monitors VM activity by polling local HTTP endpoints and reporting to a central aggregation service. It consists of three components:

1. **Monitor Agent** — a small Go binary installed on each VM that periodically queries local endpoints and reports data to the aggregation service.
2. **Aggregation Service** — a Go backend that receives reports, stores data in SQLite, and serves an API for the frontend.
3. **Frontend** — a React (TypeScript) SPA that displays monitoring data with charts and dot indicators.

---

## 2. Repository Structure

```
mx-api-monitoring/
├── agent/                        # Monitor agent (Go)
│   ├── cmd/main.go
│   ├── config/config.go
│   ├── poller/poller.go
│   ├── reporter/reporter.go
│   └── config.toml.example
├── server/                       # Aggregation service (Go)
│   ├── cmd/main.go
│   ├── config/config.go
│   ├── api/
│   │   ├── handler_report.go
│   │   ├── handler_endpoints.go
│   │   └── middleware_auth.go
│   ├── storage/
│   │   ├── db.go
│   │   └── queries.go
│   └── config.toml.example
├── frontend/                     # React + TypeScript SPA
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── api/
│   │   └── App.tsx
│   └── package.json
├── e2e/                          # End-to-end tests
│   └── ...
├── go.work                       # Go workspace (agent + server modules)
└── specs.md
```

---

## 3. Monitor Agent

### 3.1 Configuration File (TOML)

Path passed via CLI flag `--config` (default: `config.toml`).

```toml
Name = "VM1"
QueryIntervalInSeconds = 60
ReportEndpoint = "https://aaa.bbb.com/report"
ServiceApiKey = "secret-api-key"

[[endpoints]]
    Name = "VM1.Node1.nonce"
    URL = "http://127.0.0.1:8080/node/status"
    Value = "erd_nonce"
    Type = "uint64"
    NumAggregation = 100

[[endpoints]]
    Name = "VM1.Node2.nonce"
    URL = "http://127.0.0.1:8081/node/status"
    Value = "erd_nonce"
    Type = "uint64"
    NumAggregation = 100

[[endpoints]]
    Name = "VM1.Node1.epoch"
    URL = "http://127.0.0.1:8080/network/status"
    Value = "erd_epoch_number"
    Type = "uint64"
    NumAggregation = 1
```

**Config fields:**

| Field | Type | Description |
|---|---|---|
| `Name` | string | Unique identifier for this VM/agent instance |
| `QueryIntervalInSeconds` | int | How often (in seconds) to poll all endpoints |
| `ReportEndpoint` | string | Full URL of the aggregation service `/report` endpoint |
| `ServiceApiKey` | string | Shared secret sent in the `X-Api-Key` header |
| `endpoints[].Name` | string | Globally unique dot-separated name for this metric |
| `endpoints[].URL` | string | Local HTTP URL to query |
| `endpoints[].Value` | string | JSON field path to extract from the response (dot-separated for nested fields, e.g. `data.status.erd_nonce`) |
| `endpoints[].Type` | string | Data type: `"uint64"`, `"string"`, or `"bool"` |
| `endpoints[].NumAggregation` | int | Number of historical values to retain (1 = only latest) |

### 3.2 Polling Behaviour

- On startup, the agent immediately performs one poll cycle, then waits `QueryIntervalInSeconds` before the next.
- Each configured endpoint URL is queried with an HTTP GET. The response must be JSON.
- The `Value` field specifies a dot-path to extract from the JSON response (e.g. `"data.status.erd_nonce"` traverses `response["data"]["status"]["erd_nonce"]`).
- If an endpoint is unreachable or the JSON path is missing, that metric is **omitted** from the report for that cycle (not sent as error). A warning is logged locally.

### 3.3 Report Payload

After each poll cycle, the agent sends a single HTTP POST to `ReportEndpoint` with header `X-Api-Key: <ServiceApiKey>` and JSON body:

```json
{
  "metrics": {
    "VM1.Node1.nonce": {
      "value": "12345678",
      "type": "uint64",
      "numAggregation": 100
    },
    "VM1.Node2.nonce": {
      "value": "12345600",
      "type": "uint64",
      "numAggregation": 100
    },
    "VM1.Node1.epoch": {
      "value": "42",
      "type": "uint64",
      "numAggregation": 1
    },
    "VM1.Active": {
      "value": "true",
      "type": "bool",
      "numAggregation": 1
    }
  }
}
```

**Rules:**
- All values are serialized as strings in the JSON payload. The `type` field tells the server how to interpret them.
- The agent always appends `<Name>.Active` with `value = "true"`, `type = "bool"`, `numAggregation = 1`. This is the heartbeat metric.
- If the POST fails (non-2xx or network error), the agent logs an error and retries on the next poll cycle (no immediate retry).

### 3.4 Agent Binary

- Built as a single statically-linked Go binary.
- Graceful shutdown on `SIGINT` / `SIGTERM`.
- Structured JSON logging to stdout (using `log/slog`).

---

## 4. Aggregation Service

### 4.1 Configuration File (TOML)

Path passed via CLI flag `--config` (default: `config.toml`).

```toml
ListenAddress = ":8090"
ServiceApiKey = "secret-api-key"
DatabasePath = "./monitoring.db"
RetentionSeconds = 3600          # delete values older than this

[auth]
Username = "admin"
Password = "changeme"
```

**Config fields:**

| Field | Type | Description |
|---|---|---|
| `ListenAddress` | string | TCP address to bind the HTTP server to |
| `ServiceApiKey` | string | Expected value of the `X-Api-Key` header from agents |
| `DatabasePath` | string | Path to the SQLite file |
| `RetentionSeconds` | int | Values with timestamps older than this are purged |
| `auth.Username` | string | Frontend login username |
| `auth.Password` | string | Frontend login password (plaintext in config, hashed at startup for comparison) |

### 4.2 Database Schema (SQLite)

The schema is split into two tables. `metrics` holds the stable definition of each metric (its identity, type, and aggregation window). `metrics_values` holds the time-series values. This separation means the retention cleaner and the aggregation-window trimmer only ever touch `metrics_values`, so metric definitions are never silently removed — a metric disappears from the frontend only when explicitly deleted via the admin API.

**Table: `metrics`**

`name` is the natural primary key — it is already unique and is the identifier used in every API call and report payload. No surrogate id is needed.

```sql
CREATE TABLE IF NOT EXISTS metrics (
    name            TEXT    NOT NULL PRIMARY KEY,
    type            TEXT    NOT NULL,   -- 'uint64' | 'string' | 'bool'
    num_aggregation INTEGER NOT NULL DEFAULT 1
);
```

**Table: `metrics_values`**

No explicit primary key — individual value rows are never referenced by id. The three typed value columns (`value_int`, `value_str`, `value_bool`) mirror the three supported types so that the correct native SQLite type is stored and no string parsing is needed on read. Exactly one of the three will be non-NULL for any given row, determined by the parent metric's `type`. `value_bool` is stored as `INTEGER` (0/1) because SQLite has no native boolean type. `value_int` is `INTEGER`, which in SQLite is a signed 64-bit value — sufficient for all expected uint64 nonce/epoch ranges.

```sql
CREATE TABLE IF NOT EXISTS metrics_values (
    metric_name TEXT    NOT NULL REFERENCES metrics(name) ON DELETE CASCADE,
    value_int   INTEGER,            -- non-NULL when type = 'uint64'
    value_str   TEXT,               -- non-NULL when type = 'string'
    value_bool  INTEGER,            -- non-NULL when type = 'bool' (0 or 1)
    recorded_at INTEGER NOT NULL    -- Unix timestamp (seconds)
);

CREATE INDEX IF NOT EXISTS idx_metrics_values_name ON metrics_values(metric_name);
CREATE INDEX IF NOT EXISTS idx_metrics_values_recorded_at ON metrics_values(recorded_at);
```

**Storage rules:**

When a report arrives for a given metric name:

1. **Upsert the definition** — `INSERT INTO metrics (name, type, num_aggregation) VALUES (?, ?, ?) ON CONFLICT(name) DO UPDATE SET type=excluded.type, num_aggregation=excluded.num_aggregation`. This ensures the metric row always exists and stays up-to-date.
2. **Insert the new value** — insert into `metrics_values` setting only the column that matches `type`, leaving the other two NULL.
3. **Apply the aggregation window** — a single DELETE trims the window regardless of whether `num_aggregation` is 1 or greater; both cases are "keep the latest N, delete the rest":

```sql
DELETE FROM metrics_values
WHERE metric_name = ?
  AND rowid NOT IN (
      SELECT rowid FROM metrics_values
      WHERE metric_name = ?
      ORDER BY recorded_at DESC
      LIMIT ?   -- bind num_aggregation
  );
```

`rowid` is used instead of `recorded_at` in the `NOT IN` subquery to correctly handle the rare case where two rows share the same timestamp.

**Retention cleanup:**

A background goroutine runs every `max(RetentionSeconds/10, 60)` seconds and executes:

```sql
DELETE FROM metrics_values WHERE recorded_at < (strftime('%s','now') - ?);
```

This leaves all `metrics` rows intact. A metric with no remaining values will appear on the frontend with a "no data" / stale indicator rather than disappearing entirely.

### 4.3 HTTP API

All endpoints are prefixed with `/api`.

#### 4.3.1 Agent Report Endpoint

```
POST /api/report
Header: X-Api-Key: <ServiceApiKey>
Content-Type: application/json
```

Body: same payload as described in §3.3.

**Response:**
- `200 OK` with `{"ok": true}` on success.
- `401 Unauthorized` if the API key is missing or wrong.
- `400 Bad Request` if the body is malformed.

#### 4.3.2 Frontend Authentication

```
POST /api/auth/login
Content-Type: application/json
Body: {"username": "admin", "password": "changeme"}
```

**Response:**
- `200 OK` with `{"token": "<jwt>"}` on success. The JWT is signed with an HMAC-SHA256 key derived from the `ServiceApiKey` + a random salt generated at startup. Expiry: 24 hours.
- `401 Unauthorized` on bad credentials.

All other `/api/*` endpoints (except `/api/report` which uses `X-Api-Key`) require a valid `Authorization: Bearer <jwt>` header.

#### 4.3.3 List All Metrics (Latest Values)

```
GET /api/metrics
```

Returns the latest value for every distinct `name`:

```json
{
  "metrics": [
    {
      "name": "VM1.Node1.nonce",
      "value": "12345678",
      "type": "uint64",
      "numAggregation": 100,
      "recordedAt": 1708300000
    },
    {
      "name": "VM1.Active",
      "value": "true",
      "type": "bool",
      "numAggregation": 1,
      "recordedAt": 1708300000
    }
  ]
}
```

#### 4.3.4 Get Historical Values for a Metric

```
GET /api/metrics/{name}/history
```

Returns all stored rows for the given metric name, ordered by `recorded_at` ascending:

```json
{
  "name": "VM1.Node1.nonce",
  "type": "uint64",
  "numAggregation": 100,
  "history": [
    {"value": "12345500", "recordedAt": 1708299900},
    {"value": "12345600", "recordedAt": 1708299960},
    {"value": "12345678", "recordedAt": 1708300000}
  ]
}
```

#### 4.3.5 Delete a Metric

```
DELETE /api/metrics/{name}
```

Deletes all rows for the given `name`.

**Response:** `200 OK` with `{"ok": true}`.

#### 4.3.6 Static Frontend Files

The server also serves the compiled React SPA at `/` and all sub-paths (SPA fallback). The built frontend files are embedded using Go's `embed` package.

### 4.4 Service Binary

- Single statically-linked Go binary.
- Graceful shutdown on `SIGINT` / `SIGTERM` (drain in-flight requests, close DB).
- Structured JSON logging to stdout (`log/slog`).

---

## 5. Frontend (React + TypeScript)

### 5.1 Technology Stack

- **React 18** with **TypeScript**
- **Vite** as build tool
- **React Router v6** for routing
- **TanStack Query** (react-query) for data fetching and cache invalidation
- **Recharts** for time-series graphs
- **Tailwind CSS** for styling (mobile-first responsive layout)

### 5.2 Authentication

- Login page (`/login`) with username + password form.
- On success, store JWT in `localStorage`.
- All API calls include `Authorization: Bearer <token>`.
- On 401 response, redirect to `/login` and clear stored token.
- Auto-redirect from `/login` to `/` if already authenticated.

### 5.3 Pages and Layout

#### 5.3.1 Dashboard (`/`)

The main monitoring view. Metrics are fetched with polling every 30 seconds (or on-demand refresh button).

Metrics are **grouped by VM name** (the first dot-segment of the metric name, e.g. `VM1`). Within each group:

- **Bool metrics** — displayed as a small colored dot:
  - `true` → green dot
  - `false` or missing/stale → red dot
  - The `<Name>.Active` metric is displayed prominently as the VM heartbeat at the top of each group.
- **String metrics** — displayed as `name: value` text.
- **uint64 metrics with `numAggregation = 1`** — displayed as a plain number.
- **uint64 metrics with `numAggregation > 1`** — displayed as a line chart showing historical values over time (x-axis: timestamp, y-axis: value). Fetch data from `/api/metrics/{name}/history`.

A metric is considered **stale** if its `recordedAt` is older than `2 × QueryIntervalInSeconds` (the frontend does not know the agent's interval, so it uses a configurable stale threshold, defaulting to 5 minutes, shown as a warning badge).

#### 5.3.2 Admin Panel (`/admin`)

A protected page (same JWT auth) with a list of all known metric names and a delete button next to each. Intended for cleanup during testing.

- Shows a confirmation dialog before deletion.
- After deletion, the metric list refreshes.

### 5.4 Mobile Responsiveness

- Single-column layout on small screens; two-column grid on tablet; three-column on desktop for metric cards.
- Charts are responsive (use 100% container width via Recharts `ResponsiveContainer`).

---

## 6. End-to-End Tests

### 6.1 Tooling

- Go test package (`testing`) with `net/http/httptest` for server.
- The agent is invoked as a subprocess with a test config pointing to the test server.
- Tests live in `e2e/` and are tagged with `//go:build e2e`.

### 6.2 Test Scenarios

| # | Scenario | Steps | Assertion |
|---|---|---|---|
| 1 | Agent reports metrics | Start test server, start agent with test config, wait one interval | Server DB contains expected metrics with correct types and values |
| 2 | Heartbeat | After agent reports | `<Name>.Active` = `"true"` present in DB |
| 3 | NumAggregation retention | Agent reports N+1 times for a metric with `numAggregation = N` | DB has exactly N rows for that metric |
| 4 | Retention TTL | Insert metric with `recorded_at` older than retention | Background cleaner removes it |
| 5 | API key rejection | POST `/api/report` with wrong key | `401` response |
| 6 | Frontend auth | POST `/api/auth/login` with wrong password | `401`; with correct password returns JWT |
| 7 | Delete endpoint | DELETE `/api/metrics/{name}` | Metric no longer appears in GET `/api/metrics` |
| 8 | Stale endpoint skipped | Agent config has unreachable URL | That metric absent from report; other metrics present |

---

## 7. Deployment Notes

- Both binaries accept `--config <path>` for config file location.
- The aggregation service can be run behind a reverse proxy (nginx/caddy) for TLS termination. The service itself listens on plain HTTP.
- The SQLite database file should be on persistent storage. No migration tool is required for v1 — the schema is created on startup if it doesn't exist.
- Agent binaries are cross-compiled per target OS/arch. The aggregation service typically runs on a central Linux host.

---

## 8. Security Considerations

- `ServiceApiKey` must be treated as a secret (not committed to VCS). Use environment variables or a secrets manager to inject it into the config at deploy time.
- The frontend JWT secret is derived at startup from `ServiceApiKey` + random bytes; it is not persisted, so all sessions are invalidated on service restart.
- The admin password is stored in plaintext in the config file; restrict file permissions (`chmod 600`).
- All agent-to-server communication should use HTTPS in production (the `ReportEndpoint` should be an `https://` URL).

---

## 9. Out of Scope (v1)

- Multiple user accounts / RBAC.
- Alerting / notifications.
- Metric federation between aggregation services.
- Agent TLS client certificates.
- Horizontal scaling of the aggregation service (SQLite is single-writer).
