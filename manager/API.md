# BBGO Manager API

All endpoints are prefixed with `/api`. Authentication via `X-User-Id` header (set by Next.js proxy from Supabase auth).

## Health

### GET /api/health

Returns system status.

**Response:**
```json
{
  "status": "ok",
  "users": 3,
  "running": 2
}
```

## Strategies

Each user has one container (`bbgo-{userID}`) running multiple strategies.

### POST /api/users/{userID}/strategies

Add a strategy to the user's container. Auto-starts the container if stopped, restarts if already running.

**Body:**
```json
{
  "name": "BTC Grid",
  "exchange": "binance",
  "strategy": "grid2",
  "config": { "symbol": "BTCUSDT", "quantity": 0.001 },
  "mode": "paper"
}
```

**Response (201):**
```json
{
  "user_id": "...",
  "status": "running",
  "strategies": [
    {
      "id": "strat-...",
      "name": "BTC Grid",
      "exchange": "binance",
      "strategy": "grid2",
      "config": { "symbol": "BTCUSDT", "quantity": 0.001 },
      "mode": "paper"
    }
  ]
}
```

### GET /api/users/{userID}/strategies

List all strategies for a user. Returns user container state.

**Response:**
```json
{
  "user_id": "...",
  "status": "running",
  "strategies": [...]
}
```

### DELETE /api/users/{userID}/strategies/{strategyID}

Remove a strategy. If no strategies remain, stops and removes the container.

**Response:**
```json
{
  "user_id": "...",
  "status": "stopped",
  "strategies": []
}
```

## User Container Control

### POST /api/users/{userID}/start

Start the user's container. Requires at least one strategy configured.

**Response:**
```json
{
  "user_id": "...",
  "status": "running",
  "strategies": [...]
}
```

### POST /api/users/{userID}/stop

Stop and remove the user's container.

**Response:**
```json
{ "status": "stopped", "user_id": "..." }
```

### GET /api/users/{userID}/status

Get user container status and strategies.

**Response:**
```json
{
  "user_id": "...",
  "status": "running",
  "strategies": [...]
}
```

## BBGO Proxy

### ALL /api/bbgo/{userID}/*

Proxy request to the user's bbgo container API at `http://bbgo-{userID}:{BBGO_PORT}/api/*`.

The path `/api/bbgo/{userID}/sessions` is rewritten to `/api/sessions` on the container.

**Example endpoints (proxied):**

| Manager Path | Container Path | Description |
|---|---|---|
| `/api/bbgo/{userID}/sessions` | `/api/sessions` | List exchange sessions |
| `/api/bbgo/{userID}/sessions/{session}/open-orders` | `/api/sessions/{session}/open-orders` | Open orders |
| `/api/bbgo/{userID}/sessions/{session}/account` | `/api/sessions/{session}/account` | Account balances |
| `/api/bbgo/{userID}/trades` | `/api/trades` | Recent trades |
| `/api/bbgo/{userID}/ping` | `/api/ping` | Health check |

**Error (container unavailable):**
```json
{
  "error": "bot api unavailable",
  "user_id": "...",
  "details": "..."
}
```

## Backtest

### POST /api/backtest

Run a backtest via ephemeral Docker container (`docker run --rm`).

**Body:**
```json
{
  "strategy": "grid2",
  "config": { "symbol": "BTCUSDT", "exchange": "binance", "interval": "1h" },
  "start_time": "2024-01-01",
  "end_time": "2024-06-01"
}
```

**Response:**
```json
{
  "output": "... raw backtest output text ..."
}
```

## Credentials

### POST /api/credentials

Store encrypted exchange API credentials.

**Body:**
```json
{
  "exchange": "binance",
  "api_key": "...",
  "api_secret": "...",
  "passphrase": "",
  "is_testnet": false
}
```

**Response (201):**
```json
{
  "id": "cred-...",
  "user_id": "...",
  "exchange": "binance",
  "is_testnet": false
}
```

### GET /api/credentials?user_id=xxx

List credentials (keys/secrets never returned).

### DELETE /api/credentials/{id}?user_id=xxx

Delete a credential.

## Architecture

```
Frontend (Next.js :3142)
  → /api/manager/* (Next.js route handler, adds X-User-Id from auth)
    → Manager API (Go, :8090)
      ├── User strategy CRUD (in-memory + Supabase sync)
      ├── Container lifecycle (docker CLI)
      ├── Reverse proxy → bbgo containers (Docker network saas_bbgo-net)
      ├── Backtest (docker run --rm)
      └── Syncer (SQLite → Supabase every 10s)
```

Container naming: `bbgo-{userID}` (1 user = 1 container, multiple strategies)
Shared volume: `$DATA_VOLUME` mounted at `/data`
User data path: `/data/{userID}/`
