#!/usr/bin/env python3
"""Verify paper-bot data integrity against known bug patterns.

Catches the classes of silent corruption we have fixed before:
  - ROI stop-loss firing at breakeven (same-second open+close pairs)
  - Phantom CLOSE_LONG on short-only strategies
  - Orphan paper_balances rows (empty strategy_instance_id)
  - Active bots with no seeded paper_balances
  - Negative roiStopLoss percentage in saved config
  - NULL pnl on closed-position trades
  - Leverage mismatch between config and futures_config

Usage:
  python verify_paper_data.py [USER_ID] [INSTANCE_ID]

If USER_ID is omitted, reads from saas/.env (SUPABASE_* + DB_USER_ID).
If INSTANCE_ID is omitted, scans every paper bot for the user.

Exit code: 0 if all checks pass, 1 if any CRITICAL/HIGH issue is found.
"""
import json
import os
import sys
import urllib.parse
import urllib.request

ENV_PATH = os.path.join(os.path.dirname(__file__), "..", ".env")


def load_env(path):
    env = {}
    if not os.path.exists(path):
        return env
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, _, v = line.partition("=")
            env[k.strip()] = v.strip()
    return env


def sb_get(env, table, select="*", filters=None, order=None, limit=None, count=False):
    url = f"{env['SUPABASE_URL']}/rest/v1/{table}?select={select}"
    for col, val in (filters or {}).items():
        url += f"&{col}=eq.{urllib.parse.quote(str(val), safe='')}"
    if order:
        url += f"&order={order}"
    if limit:
        url += f"&limit={limit}"
    headers = {
        "apikey": env["SUPABASE_SERVICE_KEY"],
        "Authorization": f"Bearer {env['SUPABASE_SERVICE_KEY']}",
    }
    if count:
        headers["Prefer"] = "count=exact"
        req = urllib.request.Request(url, headers=headers, method="HEAD")
        with urllib.request.urlopen(req) as r:
            rng = r.headers.get("Content-Range", "*/0")
            try:
                return int(rng.split("/")[-1])
            except ValueError:
                return 0
    req = urllib.request.Request(url, headers=headers, method="GET")
    with urllib.request.urlopen(req) as r:
        return json.loads(r.read().decode("utf-8"))


SHORT_ONLY = {"pivotshort"}
WARN, FAIL = [], []


def add(level, code, msg, instance=None):
    entry = {"level": level, "code": code, "msg": msg, "instance": instance}
    (FAIL if level in ("CRITICAL", "HIGH") else WARN).append(entry)
    tag = "  WARN" if level == "WARN" else f"  {level}"
    iid = f" [{instance}]" if instance else ""
    print(f"{tag} {code}{iid}: {msg}")


def check_config(instance):
    cfg = instance.get("config") or {}
    fc = instance.get("futures_config") or {}
    iid = instance["instance_id"]

    for i, exit_method in enumerate(cfg.get("exits") or []):
        if "roiStopLoss" in exit_method:
            pct = exit_method["roiStopLoss"].get("percentage")
            try:
                pv = float(pct) if pct is not None else None
            except (TypeError, ValueError):
                pv = None
            if pv is None:
                add("HIGH", "roi_missing", f"exits[{i}].roiStopLoss.percentage is null", iid)
            elif pv < 0:
                add("HIGH", "roi_negative",
                    f"exits[{i}].roiStopLoss.percentage={pv} (negative inverts stop-loss semantics; use abs value)",
                    iid)

    if fc.get("leverage") and cfg.get("leverage") is not None:
        try:
            if float(cfg["leverage"]) != float(fc["leverage"]):
                add("MEDIUM", "leverage_mismatch",
                    f"config.leverage={cfg['leverage']} vs futures_config.leverage={fc['leverage']} "
                    f"(runtime syncs to futures_config value; stored config is misleading)",
                    iid)
        except (TypeError, ValueError):
            pass


def check_trade_patterns(env, instance, trades):
    iid = instance["instance_id"]
    if not trades:
        return

    if instance["strategy"] in SHORT_ONLY:
        close_longs = [t for t in trades if t.get("position_action") == "CLOSE_LONG"]
        if close_longs:
            add("HIGH", "phantom_long",
                f"{len(close_longs)} CLOSE_LONG trade(s) on short-only strategy "
                f"(phantom long position inherited from stale data)",
                iid)

    opens = [t for t in trades if (t.get("position_action") or "").startswith("OPEN_")]
    closes = [t for t in trades if (t.get("position_action") or "").startswith("CLOSE_")]
    paired = min(len(opens), len(closes))
    same_kline = 0
    for i in range(paired):
        o, c = opens[i], closes[i]
        try:
            same_time = (o.get("traded_at") or "")[:19] == (c.get("traded_at") or "")[:19]
            same_price = abs(float(o["price"]) - float(c["price"])) < 1e-6
            if same_time and same_price:
                same_kline += 1
        except (KeyError, TypeError, ValueError):
            pass
    if same_kline >= 3:
        add("CRITICAL", "same_kline_churn",
            f"{same_kline}/{paired} open+close pairs on same kline at same price "
            f"(ROI stop-loss firing at breakeven)",
            iid)

    null_pnl_closes = [
        t for t in closes
        if t.get("pnl") is None and t.get("position_action") != "CLOSE_LONG"
    ]
    if len(null_pnl_closes) >= 3:
        profits = sb_get(env, "paper_profits", "profit",
                         filters={"strategy_instance_id": iid}, limit=1)
        if not profits:
            add("MEDIUM", "null_pnl",
                f"{len(null_pnl_closes)} CLOSE trades with NULL pnl AND no paper_profits rows "
                f"(profit not recorded anywhere)",
                iid)


def check_balances_seeded(env, instance):
    iid = instance["instance_id"]
    uid = instance["user_id"]
    rows = sb_get(env, "paper_balances", "currency",
                  filters={"user_id": uid, "strategy_instance_id": iid})
    if not rows:
        add("HIGH", "balances_not_seeded",
            "no paper_balances row (seed-on-startup did not fire or bot never traded)",
            iid)


def check_orphan_balances(env, uid):
    rows = sb_get(env, "paper_balances", "strategy_instance_id",
                  filters={"user_id": uid})
    orphans = [r for r in rows if not r.get("strategy_instance_id")]
    if orphans:
        add("HIGH", "orphan_balances",
            f"{len(orphans)} paper_balances row(s) with empty strategy_instance_id "
            f"(left over from deleted pre-fix bots)",
            None)


def main():
    env = load_env(ENV_PATH)
    for k in ("SUPABASE_URL", "SUPABASE_SERVICE_KEY"):
        if not env.get(k):
            print(f"MISSING env: {k} in {ENV_PATH}")
            return 1

    user_id = sys.argv[1] if len(sys.argv) > 1 else env.get("DB_USER_ID")
    if not user_id:
        print("Usage: verify_paper_data.py [USER_ID] [INSTANCE_ID]")
        print("       (or set DB_USER_ID in saas/.env)")
        return 2

    target_iid = sys.argv[2] if len(sys.argv) > 2 else None

    filt = {"user_id": user_id, "mode": "paper"}
    if target_iid:
        filt["instance_id"] = target_iid
    instances = sb_get(env, "strategy_instances", "*", filters=filt)

    if not instances:
        print(f"No paper bots found for user {user_id}"
              + (f" / instance {target_iid}" if target_iid else ""))
        return 0

    print(f"Verifying {len(instances)} paper bot(s) for user {user_id}")
    print("=" * 70)

    check_orphan_balances(env, user_id)

    for inst in instances:
        iid = inst["instance_id"]
        print(f"\n[{iid}]  strategy={inst['strategy']}  symbol={inst['symbol']}")
        check_config(inst)
        check_balances_seeded(env, inst)
        trades = sb_get(env, "paper_trades",
                        "side,price,position_action,traded_at,pnl",
                        filters={"strategy_instance_id": iid},
                        order="traded_at.asc", limit=500)
        check_trade_patterns(env, inst, trades)
        print(f"  trades={len(trades)}")

    print("\n" + "=" * 70)
    crit = [e for e in FAIL if e["level"] == "CRITICAL"]
    high = [e for e in FAIL if e["level"] == "HIGH"]
    print(f"CRITICAL: {len(crit)}   HIGH: {len(high)}   WARN: {len(WARN)}")
    return 1 if FAIL else 0


if __name__ == "__main__":
    sys.exit(main())
