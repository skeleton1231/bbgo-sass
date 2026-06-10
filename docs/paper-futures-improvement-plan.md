# Paper Futures Trading Improvement Plan

## Background

Paper trading engine (`pkg/bbgo/paper_trade_exchange.go` + `paper_trade_futures.go`) supports futures and margin modes. Current issues:

1. **No direction tagging** — trades/orders/positions don't record position_action (openLong, closeShort, etc.), forcing frontend to derive direction client-side
2. **No restart recovery** — `paperFuturesState` (position amount, entry price, side) is purely in-memory; container restart resets to zero, causing incorrect position tracking
3. **No multi-instance isolation** — `paper_futures_position_risks` table unique key is `(user_id, exchange, symbol, position_side)` without `strategy_instance_id`; two strategies on same symbol collide

## Design Decisions

### 1. Position Action computed at SubmitOrder time (not fill time)

**Why:** Orders are placed by strategies that know their intent. At submission time, the paper trade engine already knows current position state + order side → can compute direction immediately.

```
SubmitOrder BUY  + no position      → openLong
SubmitOrder BUY  + long position    → addLong
SubmitOrder BUY  + short position   → closeShort (partial) or flipShortToLong
SubmitOrder SELL + no position      → openShort
SubmitOrder SELL + short position   → addShort
SubmitOrder SELL + long position    → closeLong (partial) or flipLongToShort
```

**Implication:** All three tables (`paper_orders`, `paper_trades`, `paper_positions`) get `position_action` at write time. No post-hoc UPDATE needed.

### 2. Live trading code untouched

All changes are confined to paper-only files:
- `paper_trade_exchange.go` — fill callback, RestoreFromDB
- `paper_trade_futures.go` — computePositionAction, restoreFuturesState
- New migration — paper tables only, no live table schema changes

`types.Trade`, `types.Order`, `types.PositionRisk`, `FuturesService.Insert()` — none of these are modified.

### 3. Tables affected

| Table | Change | Reason |
|-------|--------|--------|
| `paper_orders` | ADD COLUMN `position_action TEXT DEFAULT ''` | Direction known at submit time |
| `paper_trades` | ADD COLUMN `position_action TEXT DEFAULT ''` | Inherited from order at fill time |
| `paper_positions` | ADD COLUMN `position_action TEXT DEFAULT ''` | Inherited from fill |
| `paper_futures_position_risks` | ADD COLUMN `strategy_instance_id TEXT DEFAULT ''` | Multi-instance isolation |
| `paper_balances` | ADD COLUMN `strategy_instance_id TEXT DEFAULT ''` | Consistency |
| Live tables (`orders`, `trades`, etc.) | **No changes** | Live code untouched |

### 4. No FK between tables

bbgo's data model uses temporal relationships, not entity relationships. `futures_position_risks` is a current-state snapshot; `trades`/`orders`/`positions` are event records. No FK is correct.

---

## Implementation Phases

### Phase 1: Schema Migration (00034) ✅ DONE

File: `saas/web/supabase/migrations/00034_paper_futures_improvements.sql`

```sql
ALTER TABLE paper_trades ADD COLUMN IF NOT EXISTS position_action TEXT NOT NULL DEFAULT '';
ALTER TABLE paper_positions ADD COLUMN IF NOT EXISTS position_action TEXT NOT NULL DEFAULT '';
ALTER TABLE paper_futures_position_risks ADD COLUMN IF NOT EXISTS strategy_instance_id TEXT NOT NULL DEFAULT '';
ALTER TABLE paper_balances ADD COLUMN IF NOT EXISTS strategy_instance_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_paper_fpr_strategy_instance ON paper_futures_position_risks(user_id, strategy_instance_id);
```

### Phase 2: Position Action Computation

File: `pkg/bbgo/paper_trade_futures.go`

**2a.** Add `computePositionAction(state, side, quantity) string` function:

```go
func computePositionAction(state *paperFuturesState, side types.SideType, quantity fixedpoint.Value) string {
    if state == nil || state.PositionAmount.IsZero() {
        if side == types.SideTypeBuy {
            return "openLong"
        }
        return "openShort"
    }

    switch {
    case state.PositionAmount.Sign() > 0: // currently long
        if side == types.SideTypeSell {
            if quantity.Compare(state.PositionAmount) >= 0 {
                return "flipLongToShort"
            }
            return "closeLong"
        }
        return "addLong"

    case state.PositionAmount.Sign() < 0: // currently short
        if side == types.SideTypeBuy {
            if quantity.Compare(state.PositionAmount.Abs()) >= 0 {
                return "flipShortToLong"
            }
            return "closeShort"
        }
        return "addShort"
    }

    return ""
}
```

**2b.** Modify `SubmitOrder` in `paper_trade_exchange.go` to compute and store action:

In `SubmitOrder()`, before the fill happens, compute position_action using pre-fill state:

```go
var positionAction string
if isFutures {
    e.mu.Lock()
    state := e.getOrCreateFuturesState(submit.Symbol)
    positionAction = computePositionAction(state, submit.Side, submit.Quantity)
    e.mu.Unlock()
}
```

Store in `pendingPositionActions map[uint64]string` keyed by orderID.

**2c.** Propagate to trade in `buildFillLocked()`:

Look up position_action from the map using `order.OrderID`, then store it in `pendingTradeActions map[uint64]string` keyed by `trade.ID`.

**2d.** Background flush to Supabase:

Add goroutine in `StartBackgroundServices()` that runs every 3 seconds:

```go
func (e *PaperTradeExchange) flushPositionActions() {
    // Batch UPDATE paper_trades SET position_action = $1 WHERE trade_id = $2 AND user_id = $3
    // Batch UPDATE paper_positions SET position_action = $1 WHERE trade_id = $2 AND user_id = $3
}
```

### Phase 3: Container Restart Recovery

File: `pkg/bbgo/paper_trade_futures.go`

**3a.** Add `restoreFuturesStateFromDB(ctx context.Context)` method:

```go
func (e *PaperTradeExchange) restoreFuturesStateFromDB(ctx context.Context) error {
    if e.db == nil || !e.futuresSettings.IsFutures {
        return nil
    }

    rows, err := e.db.QueryxContext(ctx, fmt.Sprintf(
        "SELECT symbol, position_side, position_amount, entry_price, leverage "+
        "FROM %s WHERE user_id = $1", e.tableName("futures_position_risks")), e.userID)
    if err != nil {
        return err
    }
    defer rows.Close()

    for rows.Next() {
        var symbol, posSide, posAmt, entryPrice, leverage string
        if err := rows.Scan(&symbol, &posSide, &posAmt, &entryPrice, &leverage); err != nil {
            continue
        }
        state := e.getOrCreateFuturesState(symbol)
        state.PositionAmount = fixedpoint.MustNewFromString(posAmt)
        state.EntryPrice = fixedpoint.MustNewFromString(entryPrice)
        state.PositionSide = types.PositionType(posSide)
        if l, err := strconv.Atoi(leverage); err == nil {
            state.Leverage = l
        }
    }
    return nil
}
```

**3b.** Call from `RestoreFromDB()` in `paper_trade_exchange.go`:

After restoring balances and open orders (existing code), add:

```go
// 3. Restore futures position state from DB
if e.futuresSettings.IsFutures {
    if err := e.restoreFuturesStateFromDB(ctx); err != nil {
        log.WithError(err).Warn("paper trade: failed to restore futures state from DB")
    } else {
        log.Infof("paper trade: restored futures state from DB")
    }
}
```

### Phase 4: Tests

File: `pkg/bbgo/paper_trade_futures_test.go`

**4a.** `TestComputePositionAction` — covers all action types:
- openLong, openShort, addLong, addShort
- closeLong, closeShort
- flipLongToShort, flipShortToLong

**4b.** `TestPaperTradeExchange_RestoreFuturesStateFromDB` — verifies position state restored from DB rows

**4c.** `TestPaperTradeExchange_PositionActionPropagation` — verifies position_action flows from SubmitOrder → trade → flush

---

## Files Changed Summary

| File | Change | Phase |
|------|--------|-------|
| `saas/web/supabase/migrations/00034_paper_futures_improvements.sql` | NEW — schema migration | 1 ✅ |
| `pkg/bbgo/paper_trade_futures.go` | Add `computePositionAction`, `restoreFuturesStateFromDB`, `flushPositionActions` | 2, 3 |
| `pkg/bbgo/paper_trade_exchange.go` | Compute action in `SubmitOrder`, call restore in `RestoreFromDB`, add background flush | 2, 3 |
| `pkg/bbgo/paper_trade_futures_test.go` | New tests for position_action and restart recovery | 4 |

## Files NOT Changed (by design)

| File | Reason |
|------|--------|
| `pkg/types/position.go` | Live types untouched |
| `pkg/types/trade.go` | No PositionAction field added |
| `pkg/service/futures.go` | FuturesService.Insert() SQL unchanged |
| `pkg/bbgo/environment.go` | No live code path changes |
| `pkg/bbgo/session.go` | No session wiring changes |
| Live DB tables (`orders`, `trades`, `positions`) | No schema changes |

## Known Limitations

1. **strategy_instance_id on futures_position_risks** — Column added but not yet populated. bbgo's `FuturesService` writes without it. Can be populated later by PaperTradeExchange post-write update.

2. **position_action accuracy** — Computed from PRE-fill position state. For partially filled orders, the action reflects the state at first fill. Matches real exchange behavior.

3. **Paper table only** — `position_action` is only on `paper_*` tables. Live tables don't have it (live exchanges provide direction via their own API).
