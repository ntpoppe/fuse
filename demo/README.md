# Fuse demo

## Start

From this directory:

```bash
docker compose up --build
```

API: `http://localhost:8080`

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
| **mysql** | Pre-seeded `fuse_test.orders` (read-only `demo` user) |
| **fuse** | API in demo mode - fixed `shop` (SQLite) and `warehouse` (MySQL) connections |

Demo mode blocks adding or removing connections.

## Quick checks

Health:

```bash
curl http://localhost:8080/health
```

List connections:

```bash
curl http://localhost:8080/api/connections
```

Federated query (SQLite + MySQL):

```bash
curl -s -X POST http://localhost:8080/api/federated-query \
  -H "Content-Type: application/json" \
  -d '{"sql":"SELECT u.id, u.name, o.total, o.status FROM shop.users u INNER JOIN warehouse.orders o ON u.id = o.user_id WHERE u.active = 1 AND o.status = '\''shipped'\'' LIMIT 100"}'
```

Connection add/remove should return **403** in demo mode.

## Sample schema

| Connection | Driver | Table |
|------------|--------|-------|
| `shop` | sqlite | `users` |
| `warehouse` | mysql | `orders` |

Join key: `shop.users.id = warehouse.orders.user_id`
