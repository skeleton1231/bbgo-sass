#!/usr/bin/env bash
# Health check + smoke test for a deployed BBGO SaaS stack.
# Exits non-zero on any failure. Wire into Kubernetes liveness/readiness or
# uptime monitors (e.g. UptimeRobot, Pingdom).
set -euo pipefail

DOMAIN="${DOMAIN:-http://localhost}"
FAIL=0

probe() {
  local name="$1" url="$2" expected="${3:-200}"
  local code
  code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 10 "$url" || echo 000)"
  if [ "$code" = "$expected" ]; then
    echo "  ok  $name → $code"
  else
    echo "  ERR $name → $code (expected $expected)"
    FAIL=1
  fi
}

echo "[smoke] probing $DOMAIN"
probe "livez"        "$DOMAIN/livez"
probe "readyz"       "$DOMAIN/readyz"
probe "api-health"   "$DOMAIN/api/health"
probe "homepage"     "$DOMAIN/"
probe "robots.txt"   "$DOMAIN/robots.txt"
probe "static-404"   "$DOMAIN/_next/static/nonexistent.js" 404

# Internal probes must NOT be exposed externally.
probe "metrics-blocked" "$DOMAIN/metrics" 404

if [ $FAIL -ne 0 ]; then
  echo "[smoke] FAILED"
  exit 1
fi
echo "[smoke] all checks passed"
