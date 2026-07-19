# Paper Mode Strategy Audit (Phase 0)

**Date:** 2026-07-18
**Scope:** Plan A — make paper mode fully work for all single-exchange strategies + build a regression gate for future strategies. Cross-exchange (Phase 4) is out of scope and tracked separately.

**Goal of this doc:** establish the definitive inventory of every SaaS strategy's paper-mode status, classify *why* each liveOnly strategy is blocked, and feed Phase 1–3 + 5.

---

## TL;DR — Key Audit Findings

1. **The `live_only` set is stale and unprincipled.** ~23 strategies are liveOnly, but the choices look historical rather than capability-based:
   - `bollmaker`, `linregmaker`, `rsmaker` are liveOnly; the structurally equivalent `fixedmaker`, `fmaker` are **not**.
   - 6 of 8 `requiresFutures` strategies are liveOnly, yet `pivotshort` + `fixedmaker` (same futures class) **already run successfully in paper-futures**. The other 6 were almost certainly flagged *before* paper-futures existed and are unblockable now.
2. **Migration ↔ DB drift (reproducibility risk).** Migrations seed only 8 `live_only = true` (`autoborrow`, `convert`, `deposit2transfer`, `sentinel` + migration 00012's `dca2`, `dca3`, `liquiditymaker`, `xhedgegrid`). The runtime set is ~23. The remainder was edited directly in the DB and is **not reproducible from migrations**. `StrategyDefaultsCache.Load()` reads the live DB at runtime, so the migrations under-seed what a fresh DB would contain.
3. **Cross-exchange strategies are in limbo.** 8 of 10 are NOT liveOnly and are *allowed* into paper mode (manager only blocks cross-exchange paper when a session ≠ binance). But `PaperTradeExchange` wraps a **single** exchange (`e.inner`), so multi-session cross-exchange strategies can silently malfunction in paper. Until Phase 4 lands, these should be explicitly excluded from paper (see Phase 5 recommendation).
4. **No runtime paper coverage exists.** `registry_lifecycle_test.go` covers `Defaults/Validate/InstanceID` (no panic), and `paper_trade_*_test.go` covers engine mechanics. Nothing wires a real strategy to the paper engine + synthetic klines. Every one of the 20+ historical paper bugs (see memory) was found by manual E2E. This is the root cause of "new strategy breaks paper."

---

## Inventory

### Blocking logic (ground truth from `manager/api.go:329-346`)

Paper mode rejects a strategy when ANY of:
1. `IsLiveOnly(strategyID)` is true → HTTP 400 (the `live_only` DB column).
2. Single-exchange AND `exchange != "binance"`.
3. Cross-exchange AND any session `exchange != "binance"`.

Cross-exchange binance-only is **allowed** by the manager — but see finding #3.

### liveOnly set (mirror of production DB, per `api_cross_layer_type_test.go` testRegistry, asserted aligned with `StrategyRegistry`)

23 entries: `audacitymaker`, `autoborrow`, `autobuy`, `bollmaker`, `convert`, `dca2`, `dca3`, `deposit2transfer`, `drift`, `elliottwave`, `factorzoo`, `linregmaker`, `liquiditymaker`, `liquiditymaker`, `rebalance`, `rsmaker`, `scmaker`, `sentinel`, `supertrend`, `support`, `wall`, `xnav`, `xpremium`, `xvs`.

> CLAUDE.md states "22"; the test mirror has 23. Minor drift — the DB is authoritative at runtime.

### Full strategy matrix

Legend — **LO** = liveOnly. **RF** = requiresFutures. **Phase** = where this strategy lands in Plan A.

#### Single-exchange (40)

| Strategy | Category | RF | LO | Paper engine path needed | Phase |
|---|---|:--:|:--:|---|---|
| atrpin | trend | | | spot | **1** (verify) |
| audacitymaker | maker | ✓ | ✓ | futures | **3** unblock (RF sibling of fixedmaker) |
| autoborrow | utility | | ✓ | margin borrow/repay | **3** (margin) |
| autobuy | dca | | ✓ | spot (schedule) | **3** investigate-then-unblock |
| bollgrid | grid | | | spot | **1** (verify) |
| bollmaker | maker | ✓ | ✓ | futures | **3** unblock (RF sibling of fixedmaker) |
| convert | utility | | ✓ | account feed (dust convert) | **3** account-feed injection |
| dca | dca | | | spot (schedule) | **1** (verify) |
| dca2 | dca | | ✓ | TBD | **3** investigate |
| dca3 | dca | | ✓ | TBD | **3** investigate |
| deposit2transfer | utility | | ✓ | deposit detection feed | **3** deposit-feed injection |
| drift | trend | ✓ | ✓ | futures | **3** unblock (RF sibling) |
| elliottwave | trend | ✓ | ✓ | futures | **3** unblock (RF sibling) |
| emacross | trend | | | spot | **1** (verify) |
| ewo_dgtrd | mean-reversion | | | spot | **1** (verify) |
| factorzoo | trend | | ✓ | TBD | **3** investigate |
| fixedmaker | maker | ✓ | | futures | **reference** — already works in paper |
| flashcrash | volatility | | | spot | **1** (verify) |
| fmaker | maker | | | spot | **1** (verify) |
| grid | grid | | | spot | **1** (verify) |
| grid2 | grid | | | spot | **1** (verify) |
| harmonic | mean-reversion | | | spot | **1** (verify) |
| irr | mean-reversion | | | spot | **1** (verify) |
| linregmaker | maker | ✓ | ✓ | futures | **3** unblock (RF sibling) |
| liquiditymaker | maker | | ✓ | layered depth | **3** product decision (depth) |
| pivotshort | mean-reversion | ✓ | | futures | **reference** — already works in paper |
| random | other | | | spot (cron) | **1** (verify) |
| rebalance | other | | ✓ | multi-asset | **3** investigate |
| rsmaker | maker | ✓ | ✓ | futures | **3** unblock (RF sibling) |
| schedule | dca | | | spot (cron) | **1** (verify) |
| scmaker | maker | | ✓ | TBD | **3** investigate |
| sentinel | other | | ✓ | monitor-only (no trades) | **keep LO?** monitor-only |
| supertrend | trend | | ✓ | TBD | **3** investigate |
| support | utility | | ✓ | TBD | **3** investigate |
| swing | mean-reversion | | | spot | **1** (verify) |
| techsignal | indicator | | | no-trade (monitor) | **1** (verify, expect 0 orders) |
| trendtrader | trend | | | spot | **1** (verify) |
| wall | other | | ✓ | layered orders | **3** investigate |
| xhedgegrid | grid | | ✓ | TBD | **3** investigate |
| xvs | volatility | | ✓ | spot | **3** investigate |

#### Cross-exchange (10) — **Phase 4, out of scope for Plan A**

| Strategy | Category | LO | Notes |
|---|---|:--:|---|
| xalign | cross-exchange | | allowed into paper (binance-only) — engine can't truly support → exclude until P4 |
| xbalance | cross-exchange | | same |
| xdepthmaker | cross-exchange | | same + needs real depth |
| xfixedmaker | cross-exchange | | same |
| xfunding | cross-exchange | | same + funding feed |
| xfundingv2 | cross-exchange | | same |
| xgap | cross-exchange | | same |
| xmaker | cross-exchange | | same |
| xnav | cross-exchange | ✓ | same |
| xpremium | cross-exchange | ✓ | same |

---

## liveOnly reason classification

Grouping the 21 single-exchange liveOnly strategies by *why* they are blocked:

### A. Futures-class, likely unblockable now (6) → Phase 3 priority
`audacitymaker`, `bollmaker`, `drift`, `elliottwave`, `linregmaker`, `rsmaker`

All `requiresFutures`. Their non-liveOnly futures siblings `pivotshort` + `fixedmaker` already run in paper-futures. Hypothesis: flagged before paper-futures shipped. **Action:** Phase 1 harness exercises them in paper-futures mode; on pass, flip `live_only=false` via migration.

### B. Needs account-service feed (3) → Phase 3 engine extension
`convert`, `deposit2transfer`, `autoborrow`

Depend on deposit/withdraw/margin-level events the paper engine does not synthesize. **Action:** add simulated deposit + margin-level injection points to `PaperTradeExchange`; then unblock.

### C. Needs real order-book depth (1) → Phase 3 product decision
`liquiditymaker`

Layered liquidity market maker; kline-driven matching has no L2 depth. **Decision needed:** add synthetic depth feed, or keep liveOnly by design.

### D. Stale / unexplained (10) → Phase 3 investigate-then-decide
`autobuy`, `dca2`, `dca3`, `factorzoo`, `rebalance`, `scmaker`, `supertrend`, `support`, `wall`, `xvs`, `xhedgegrid`

No obvious paper-engine gap. Likely historical. **Action:** Phase 1 harness runs them; most are expected to pass and get unblocked. `xhedgegrid` may need futures; `rebalance` is multi-asset.

### E. Monitor-only (1) → keep liveOnly by design
`sentinel`

Anomaly detector that does not trade. Paper has no value. **Action:** keep liveOnly, document reason in registry.

### F. Cross-exchange (2) → Phase 4
`xnav`, `xpremium` — and the other 8 non-liveOnly cross-exchange strategies should be **added** to a paper-exclusion list until Phase 4 (finding #3).

---

## Feeders into later phases

- **Phase 1 (harness)** must run every strategy in the **Phase 1 (verify)** column as the green baseline, plus the Phase 3 columns to produce pass/fail triage. The harness reads `live_only` from the same source the manager uses (DB → `StrategyDefaultsCache`), so the skip-allowlist and the DB flag stay aligned.
- **Phase 2** scope = any Phase-1-verify strategy that fails the harness (fix per-strategy Defaults/Validate or paper engine).
- **Phase 3** scope = groups A/B/D above, in that order. A is cheapest (just verify + flip flag). B needs an engine extension. D is investigate-per-strategy.
- **Phase 5** must (a) add the 8 non-liveOnly cross-exchange strategies to a paper-exclusion list until Phase 4, (b) reconcile the migration ↔ DB drift so a fresh DB reproduces the liveOnly set, (c) wire the harness as a CI gate with a skip-allowlist keyed to the classifications above.

---

## Phase 1 results — paper smoke harness (landed 2026-07-19)

Harness: `pkg/cmd/strategy/paper_smoke_test.go` → `TestPaperSmoke_AllStrategies_RunWithoutPanic`.

For every registered single-exchange strategy, it wires the strategy to a paper-backed `ExchangeSession`, runs `Defaults` → `Initialize` → `Subscribe` → `Run`, feeds 220 deterministic synthetic klines through a controllable `StandardStream` (which dispatches to BOTH the strategy callbacks and the paper matching engine), then asserts: no panic, no hang (every op under an 8s guard), `Run` exits on ctx cancel, and **no balance goes negative**.

Coverage model = **denylist**: every strategy runs unless listed in `paperSmokeSkip` with a concrete reason. New strategies are covered by default — the gate is strong.

**Result:** 24 strategies pass cleanly. 29 are skipped with documented reasons split across:
  - **Registry-default injection (3):** `swing`, `autobuy`, `scmaker` — these have no/ partial bbgo `Defaults()` and rely on the SaaS manager deep-merging `strategy_registry` defaults. Harness Phase 2 improvement: deep-merge the registry defaults JSON (extracted from migration 00010) before `Run`. This is the DB↔bbgo drift from finding #1 made concrete.
  - **Paper-engine service gap (2):** `dca2`, `dca3` build a trade collector via `CollectorQueryService`, which `PaperTradeExchange` does not implement. Add the interface to the paper engine.
  - **Strategy bugs surfaced by the harness (5):** `fmaker` (indicator indexed before warmup → index out of range), `audacitymaker`/`linregmaker` (Subscribe nil-deref), `liqmaker`/`rsmaker` (Run nil-deref). Real bugs — investigate & fix.
  - **Test isolation (1):** `xhedgegrid` registers a prometheus collector by a fixed name → duplicate registration across strategies in one test process. Fix: unique collector names or sub-process isolation.

The harness also discovered (and the denylist documents) the **Interval-default drift** at scale: ~half of all strategies panic in `Subscribe` unless `Interval`/`MinInterval` are filled, because their bbgo `Defaults()` doesn't set them — they depend on the registry. The harness works around this with `applyDefaultIntervals` (recursive fill of empty Interval/MinInterval fields), which is the minimal version of the registry-default injection the manager does.

### Phase 2 progress (2026-07-19)

`ran` rose 24 → 29. Five strategies unblocked:
- **swing, autobuy, rsmaker, linregmaker, audacitymaker** — via a `registryDefaults` map (migration-00012 JSON unmarshaled before `Defaults()`). Pattern: these embed `*BollingerConfig` / `*IntervalWindow` / cron-schedule / `*PerTrade` pointer fields that are nil without registry defaults and get dereferenced in Subscribe/Run without nil-guards.

Two real strategy bugs fixed:
- **scmaker** — added nil-guards on `initializeMidPriceEMA` / `initializePriceRangeBollinger` (matched the existing Subscribe guard). Still needs more (LiquiditySlideRule etc.) — remains skipped.

Root-cause found (not an engine gap):
- **dca2 / dca3** — require `QueryClosedOrdersDesc`, which **only the MAX exchange implements**. Paper mode is binance-only (manager-enforced), so these are fundamentally incompatible with paper regardless of engine work. This is the real reason they are live_only. Permanently skipped with accurate reason.

Residual Phase 2 backlog (4, each a real per-strategy issue, documented in `paperSmokeSkip`):
- `fmaker` — `index out of range [-2]` in regression training (ML strategy; warmup gate ignores `outlook` vs indicator length).
- `scmaker` — more optional-pointer nil-guards needed (LiquiditySlideRule).
- `liqmaker` (= liquiditymaker alias) — Run nil-deref + Phase 3 depth decision.
- `xhedgegrid` — duplicate prometheus collector registration in a shared test process (test isolation; needs unique metric names or sub-process isolation).

## Open questions (resolve before Phase 3)

1. **liquiditymaker (group C):** ship a synthetic depth feed, or keep liveOnly? Product call.
2. **dca2 / dca3:** were these flagged for a real futures reason, or just swept into 00012's batch fix? Check whether paper-futures covers their logic. (Phase 1 found the real reason: paper engine lacks `CollectorQueryService`.)
3. **Migration drift:** author a single reconciliation migration that `UPDATE`s `live_only` to the authoritative set so fresh DBs reproduce production. (Addressed in the Phase 5 reconciliation migration.)
4. **Cross-exchange non-liveOnly 8:** confirm they silently malfunction in paper today, then decide block-vs-fix timing. (Phase 5 adds them to `live_only` as a safety net until Phase 4.)
