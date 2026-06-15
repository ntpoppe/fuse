# Fuse demo

Docker stack: **web** (static UI) + **fuse** (JSON API) + **mysql** (seed data).

The UI and API are separate services. The browser loads the UI from one origin and calls the API on another.

| URL | Service |
|-----|---------|
| http://localhost:8080 | Web UI (`demo/ui/`) |
| http://localhost:5000 | JSON API |

## Start

From this directory:

```bash
docker compose up --build
```

Open **http://localhost:8080** in a browser.

Stop with `Ctrl+C`, then remove containers:

```bash
docker compose down
```

Reset demo data (remove MySQL volume):

```bash
docker compose down -v
```

## What starts

| Service | Role |
|---------|------|
| **web** | Apache httpd: serves static files from `demo/ui/` on `:8080` |
| **fuse** | JSON API in demo mode on `:5000` (CORS allows `http://localhost:8080`) |
| **mysql** | Pre-seeded `fuse_test.orders` (read-only `demo` user) |

Demo mode blocks adding or removing connections on the API.

## Quick checks

API health:

```bash
curl http://localhost:5000/health
```

List connections:

```bash
curl http://localhost:5000/api/connections
```

Federated query (SQLite + MySQL). Omit `id` for federated; include `id` for single-connection queries:

```bash
curl -s -X POST http://localhost:5000/api/query \
  -H "Content-Type: application/json" \
  -d '{"sql":"SELECT u.id, u.name, o.total, o.status FROM shop.users u INNER JOIN warehouse.orders o ON u.id = o.user_id WHERE u.active = 1 AND o.status = '\''shipped'\'' LIMIT 100"}'
```

Single-connection query:

```bash
curl -s -X POST http://localhost:5000/api/query \
  -H "Content-Type: application/json" \
  -d '{"id":"shop","sql":"SELECT id, name FROM users WHERE active = 1 LIMIT 25"}'
```

Connection add/remove should return **403** in demo mode.

## Sample schema

| Connection | Driver | Table | Rows |
|------------|--------|-------|------|
| `shop` | sqlite | `users` | 25 |
| `warehouse` | mysql | `orders` | 47 |

**shop.users:** id, name, email, active, country, tier, created_at

**warehouse.orders:** id, user_id, product, quantity, total, status, channel, ordered_at

Join key: `shop.users.id = warehouse.orders.user_id`

After changing seed SQL, reset MySQL data: `docker compose down -v` then `docker compose up --build`.

## Web UI

Static files: `demo/ui/`. Edit and refresh the browser. No Go rebuild needed.

Point the UI at the API in `demo/ui/config.js`:

```js
window.FUSE_API_BASE = "http://localhost:5000";
```

When hosting UI and API on different domains in production, update `config.js` and set `FUSE_CORS_ORIGINS` (or `-cors-origins`) on the API to include both `http://localhost:8080` and `http://127.0.0.1:8080` if you use either URL.

## API only (no Docker UI)

Run the API:

```bash
go run ./cmd/server -demo \
  -cors-origins "http://localhost:8080" \
  -demo-sqlite-path ./shop.db \
  -demo-mysql-dsn "demo:demo@tcp(127.0.0.1:3306)/fuse_test"
```

Serve the UI from `demo/ui/` with any static file server on another port (e.g. `npx serve demo/ui -p 8080`).
