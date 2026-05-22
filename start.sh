#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# BBGO SaaS Startup Script (Windows Git Bash / Linux / macOS)
# ============================================================

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BBGO_DIR="$ROOT_DIR"
SAAS_DIR="$ROOT_DIR/saas"
WEB_DIR="$SAAS_DIR/web"
MANAGER_DIR="$SAAS_DIR/manager"
BBGO_BIN="$BBGO_DIR/build/bbgo/bbgo-slim"
MANAGER_BIN="$SAAS_DIR/manager/manager"
PID_DIR="$SAAS_DIR/.pids"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log()  { echo -e "${GREEN}[START]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
err()  { echo -e "${RED}[ERROR]${NC} $1"; }

mkdir -p "$PID_DIR"

# Detect OS
case "$(uname -s)" in
  MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
  Linux*)               OS="linux" ;;
  Darwin*)              OS="darwin" ;;
  *)                    OS="unknown" ;;
esac

# --------------------------------------------------
# 1. Build bbgo-slim if not exists
# --------------------------------------------------
if [ ! -f "$BBGO_BIN" ]; then
  log "Building bbgo-slim..."
  mkdir -p "$BBGO_DIR/build/bbgo"
  (cd "$BBGO_DIR" && go build -tags release -o "$BBGO_BIN" ./cmd/bbgo)
  log "bbgo-slim built: $BBGO_BIN"
else
  log "bbgo-slim already exists: $BBGO_BIN"
fi

# --------------------------------------------------
# 2. Build Go Manager if not exists
# --------------------------------------------------
if [ ! -f "$MANAGER_BIN" ]; then
  log "Building BBGO Manager..."
  (cd "$MANAGER_DIR" && GOPROXY=https://goproxy.cn,direct go build -o "$MANAGER_BIN" .)
  log "Manager built: $MANAGER_BIN"
else
  log "Manager already exists: $MANAGER_BIN"
fi

# --------------------------------------------------
# 3. Install web dependencies if needed
# --------------------------------------------------
if [ ! -d "$WEB_DIR/node_modules" ]; then
  log "Installing web dependencies..."
  (cd "$WEB_DIR" && npm install)
else
  log "Web dependencies ready"
fi

# --------------------------------------------------
# 4. Start Manager
# --------------------------------------------------
start_manager() {
  if [ -f "$PID_DIR/manager.pid" ] && kill -0 "$(cat "$PID_DIR/manager.pid")" 2>/dev/null; then
    warn "Manager already running (PID $(cat "$PID_DIR/manager.pid"))"
    return
  fi

  log "Starting BBGO Manager on :8090..."
  (cd "$MANAGER_DIR" && BBGO_BINARY="$BBGO_BIN" "$MANAGER_BIN") &
  echo $! > "$PID_DIR/manager.pid"
  sleep 1

  if kill -0 "$(cat "$PID_DIR/manager.pid")" 2>/dev/null; then
    log "Manager started (PID $(cat "$PID_DIR/manager.pid"))"
  else
    err "Manager failed to start. Check logs above."
  fi
}

# --------------------------------------------------
# 5. Start Next.js dev server
# --------------------------------------------------
start_web() {
  if [ -f "$PID_DIR/web.pid" ] && kill -0 "$(cat "$PID_DIR/web.pid")" 2>/dev/null; then
    warn "Web dev server already running (PID $(cat "$PID_DIR/web.pid"))"
    return
  fi

  log "Starting Next.js dev server on :3142..."
  (cd "$WEB_DIR" && npx next dev --port 3142) &
  echo $! > "$PID_DIR/web.pid"
  sleep 2

  if kill -0 "$(cat "$PID_DIR/web.pid")" 2>/dev/null; then
    log "Web dev server started (PID $(cat "$PID_DIR/web.pid"))"
  else
    err "Web dev server failed to start."
  fi
}

# --------------------------------------------------
# 6. Stop all services
# --------------------------------------------------
stop_all() {
  log "Stopping all services..."
  for svc in manager web; do
    if [ -f "$PID_DIR/$svc.pid" ]; then
      local pid
      pid=$(cat "$PID_DIR/$svc.pid")
      if kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null && log "$svc stopped (PID $pid)" || warn "$svc already stopped"
      fi
      rm -f "$PID_DIR/$svc.pid"
    fi
  done
}

# --------------------------------------------------
# Main
# --------------------------------------------------
case "${1:-start}" in
  start)
    start_manager
    start_web
    echo ""
    log "All services running:"
    log "  Manager API:  http://localhost:8090"
    log "  Web Frontend: http://localhost:3142"
    echo ""
    log "Press Ctrl+C to stop all services"
    trap stop_all EXIT INT TERM
    wait
    ;;
  stop)
    stop_all
    ;;
  restart)
    stop_all
    sleep 1
    start_manager
    start_web
    echo ""
    log "All services restarted."
    trap stop_all EXIT INT TERM
    wait
    ;;
  build)
    log "Rebuilding all..."
    rm -f "$BBGO_BIN" "$MANAGER_BIN"
    mkdir -p "$BBGO_DIR/build/bbgo"
    (cd "$BBGO_DIR" && go build -tags release -o "$BBGO_BIN" ./cmd/bbgo)
    (cd "$MANAGER_DIR" && GOPROXY=https://goproxy.cn,direct go build -o "$MANAGER_BIN" .)
    log "Build complete."
    ;;
  status)
    for svc in manager web; do
      if [ -f "$PID_DIR/$svc.pid" ] && kill -0 "$(cat "$PID_DIR/$svc.pid")" 2>/dev/null; then
        log "$svc: running (PID $(cat "$PID_DIR/$svc.pid"))"
      else
        warn "$svc: stopped"
      fi
    done
    ;;
  *)
    echo "Usage: $0 {start|stop|restart|build|status}"
    exit 1
    ;;
esac
