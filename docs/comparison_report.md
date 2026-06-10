# BBGO SaaS vs Original BBGO — Comprehensive Comparison Report

**Date**: 2026-06-10
**Our Fork**: `D:\git_projects\bbgo` (skeleton1231/bbgo fork)
**Original**: `D:\git_projects\compare\bbgo` (c9s/bbgo upstream)

---

## Part 1: Schema Comparison — Supabase vs MySQL

### Column Mapping Summary

Our Supabase schema uses `TEXT` for all numeric fields (bbgo uses `DECIMAL(16,8)`) and adds SaaS-specific columns (`user_id`, `id UUID`, `strategy_instance_id`, `position_action`). This is intentional — TEXT allows arbitrary precision from fixedpoint.Value. The key concern is **which columns exist**, not their PostgreSQL types.

---

### 1.1 `orders` table

| Column | MySQL (original) | Supabase (ours) | Status |
|--------|:---:|:---:|:---:|
| `gid` BIGINT AUTO_INCREMENT | YES | YES (BIGSERIAL, added in 00030) | OK |
| `exchange` VARCHAR(24) | YES | YES TEXT | OK |
| `order_id` BIGINT UNSIGNED | YES | YES TEXT | OK — stored as string in PG |
| `client_order_id` VARCHAR(122) | YES | YES TEXT | OK |
| `order_type` VARCHAR(16) | YES | YES TEXT | OK |
| `symbol` VARCHAR(32) | YES | YES TEXT | OK |
| `status` VARCHAR(12) | YES | YES TEXT | OK |
| `time_in_force` VARCHAR(4) | YES | YES TEXT | OK |
| `price` DECIMAL(16,8) | YES | YES TEXT | OK |
| `stop_price` DECIMAL(16,8) | YES | YES TEXT | OK |
| `quantity` DECIMAL(16,8) | YES | YES TEXT | OK |
| `executed_quantity` DECIMAL(16,8) | YES | YES TEXT | OK |
| `side` VARCHAR(4) | YES | YES TEXT | OK |
| `is_working` BOOL | YES | YES BOOLEAN | OK |
| `created_at` DATETIME(3) | YES | YES TIMESTAMPTZ | OK |
| `updated_at` DATETIME(3) | YES | YES TIMESTAMPTZ | OK |
| `is_margin` BOOLEAN | YES | YES BOOLEAN | OK |
| `is_isolated` BOOLEAN | YES | YES BOOLEAN | OK |
| `is_futures` BOOLEAN | YES | YES BOOLEAN | OK |
| `uuid` VARBINARY(36) | YES | YES TEXT (as `order_uuid`) | MISMATCH |
| `actual_order_id` BIGINT | YES | YES BIGINT | OK |
| `strategy_instance_id` | NO | YES | SaaS extension |
| `position_action` | NO | YES | SaaS extension |
| `user_id` | NO | YES | SaaS extension (multi-tenant) |
| `id` UUID | NO | YES | SaaS extension (PK) |

**Issue 1.1.A — `uuid` vs `order_uuid` column name**: The original MySQL uses column name `uuid` (stored as VARBINARY(36) using UUID_TO_BIN). Our PostgreSQL INSERT uses `order_uuid` (TEXT). The Go type `Order.UUID` has `db:"uuid"` tag but the postgres Insert maps it to column `order_uuid`. This works because the postgres INSERT uses a manual parameter map, not StructScan. However, SELECT queries using `SELECT *` would try to map the `order_uuid` column to `Order.UUID` via the `db:"uuid"` tag — **this would FAIL** because the column name doesn't match the tag. The `genTradeSelectColumns` function only applies column remapping for MySQL driver. For postgres, it uses `SELECT *` which would return `order_uuid` but StructScan expects `uuid`.

**Impact**: Reading orders back from Supabase would fail to populate the `UUID` field. This may not be critical (UUID is not heavily used), but it's a correctness issue.

---

### 1.2 `trades` table

| Column | MySQL (original) | Supabase (ours) | Status |
|--------|:---:|:---:|:---:|
| `gid` BIGINT AUTO_INCREMENT | YES | YES BIGSERIAL (00030) | OK |
| `id` BIGINT UNSIGNED | YES | YES (stored as `trade_id` TEXT) | MISMATCH |
| `order_id` BIGINT UNSIGNED | YES | YES TEXT | OK |
| `order_uuid` VARBINARY(16) | YES | YES TEXT | OK |
| `exchange` VARCHAR(24) | YES | YES TEXT | OK |
| `price` DECIMAL(16,8) | YES | YES TEXT | OK |
| `quantity` DECIMAL(16,8) | YES | YES TEXT | OK |
| `quote_quantity` DECIMAL(16,8) | YES | YES TEXT | OK |
| `symbol` VARCHAR(32) | YES | YES TEXT | OK |
| `side` VARCHAR(4) | YES | YES TEXT | OK |
| `is_buyer` BOOLEAN | YES | YES BOOLEAN | OK |
| `is_maker` BOOLEAN | YES | YES BOOLEAN | OK |
| `traded_at` DATETIME(3) | YES | YES TIMESTAMPTZ | OK |
| `fee` DECIMAL(16,8) | YES | YES TEXT | OK |
| `fee_currency` VARCHAR(16) | YES | YES TEXT | OK |
| `is_margin` BOOLEAN | YES | YES BOOLEAN | OK |
| `is_futures` BOOLEAN | YES | YES BOOLEAN | OK |
| `is_isolated` BOOLEAN | YES | YES BOOLEAN | OK |
| `strategy` VARCHAR(32) | YES | YES TEXT | OK |
| `pnl` DECIMAL | YES | YES TEXT | OK |
| `strategy_instance_id` | NO | YES | SaaS extension |
| `position_action` | NO | YES | SaaS extension |
| `user_id` | NO | YES | SaaS extension (multi-tenant) |
| `id` UUID | NO | YES | SaaS extension (PK) |

**Issue 1.2.A — `id` vs `trade_id` column name**: The original MySQL uses `id` as the column name. Our Supabase uses `trade_id` (TEXT). The Go struct `Trade` has `ID uint64 db:"id"`. The postgres Insert explicitly maps `trade.ID` to `trade_id` column. For SELECT, `genTradeSelectColumns` only remaps for MySQL driver. For postgres, `SELECT *` returns `trade_id` but StructScan expects `id`. **UUID and trade ID fields won't populate via StructScan when reading from postgres**.

**Issue 1.2.B — Missing `inserted_at` column**: Original bbgo `Trade` has `InsertedAt *Time db:"inserted_at"`. This column is not in the Supabase schema. Not a problem for inserts, but if any code reads it back, it will be NULL.

---

### 1.3 `positions` table

| Column | MySQL (original) | Supabase (ours) | Status |
|--------|:---:|:---:|:---:|
| `gid` BIGINT AUTO_INCREMENT | YES | NO | MISSING |
| `strategy` VARCHAR(32) | YES | YES TEXT | OK |
| `strategy_instance_id` VARCHAR(64) | YES | YES TEXT | OK |
| `symbol` VARCHAR(32) | YES | YES TEXT | OK |
| `quote_currency` VARCHAR(10) | YES | YES TEXT | OK |
| `base_currency` VARCHAR(16) | YES | YES TEXT | OK |
| `average_cost` DECIMAL(16,8) | YES | YES TEXT | OK |
| `base` DECIMAL(16,8) | YES | YES TEXT | OK |
| `quote` DECIMAL(16,8) | YES | YES TEXT | OK |
| `profit` DECIMAL(16,8) | YES | YES TEXT | OK |
| `net_profit` DECIMAL(16,8) | NO (not in original positions DDL) | YES | Our extension |
| `trade_id` BIGINT UNSIGNED | YES | YES BIGINT | OK |
| `side` VARCHAR(4) | YES | YES TEXT | OK |
| `exchange` VARCHAR(20) | YES | YES TEXT | OK |
| `traded_at` DATETIME(3) | YES | YES TIMESTAMPTZ | OK |
| `position_action` | NO | YES | SaaS extension |
| `user_id` | NO | YES | SaaS extension |
| `id` UUID | NO | YES | SaaS extension (PK) |

**Issue 1.3.A — No `gid` column in positions**: The positions table has no `gid` BIGSERIAL. Orders/trades got gid in migration 00030, but positions did not. The Go struct `Position` does not have a `gid` field (unlike Order/Trade), so this may not matter for writes. But any future query that uses `gid` for ordering will not work.

**Note**: Original MySQL positions table does NOT have `net_profit` — it is our extension. The Insert code in our fork passes `netProfit` but original bbgo does not have this column either. This works because postgres ignores extra params in the map that do not match columns... actually no — it would fail because the INSERT includes `net_profit` in the column list and it IS in the Supabase table. This is fine — it is an extension that is properly supported.

---

### 1.4 `profits` table

| Column | MySQL (original) | Supabase (ours) | Status |
|--------|:---:|:---:|:---:|
| `gid` BIGINT AUTO_INCREMENT | YES | NO | MISSING |
| `strategy` VARCHAR(32) | YES | YES TEXT | OK |
| `strategy_instance_id` VARCHAR(64) | YES | YES TEXT | OK |
| `symbol` VARCHAR(32) | YES | YES TEXT | OK |
| `average_cost` DECIMAL(16,8) | YES | YES TEXT | OK |
| `profit` DECIMAL(16,8) | YES | YES TEXT | OK |
| `net_profit` DECIMAL(16,8) | YES | YES TEXT | OK |
| `profit_margin` DECIMAL(16,8) | YES | YES TEXT | OK |
| `net_profit_margin` DECIMAL(16,8) | YES | YES TEXT | OK |
| `quote_currency` VARCHAR(10) | YES | YES TEXT | OK |
| `base_currency` VARCHAR(16) | YES | YES TEXT | OK |
| `exchange` VARCHAR(24) | YES | YES TEXT | OK |
| `is_futures` BOOLEAN | YES | YES BOOLEAN | OK |
| `is_margin` BOOLEAN | YES | YES BOOLEAN | OK |
| `is_isolated` BOOLEAN | YES | YES BOOLEAN | OK |
| `trade_id` BIGINT UNSIGNED | YES | YES BIGINT | OK |
| `side` VARCHAR(4) | YES | YES TEXT | OK |
| `is_buyer` BOOLEAN | YES | YES BOOLEAN | OK |
| `is_maker` BOOLEAN | YES | YES BOOLEAN | OK |
| `price` DECIMAL(16,8) | YES | YES TEXT | OK |
| `quantity` DECIMAL(16,8) | YES | YES TEXT | OK |
| `quote_quantity` DECIMAL(16,8) | YES | YES TEXT | OK |
| `traded_at` DATETIME(3) | YES | YES TIMESTAMPTZ | OK |
| `fee_in_usd` DECIMAL(16,8) | YES | YES TEXT | OK |
| `fee` DECIMAL(16,8) | YES | YES TEXT | OK |
| `fee_currency` VARCHAR(16) | YES | YES TEXT | OK |
| `user_id` | NO | YES | SaaS extension |
| `id` UUID | NO | YES | SaaS extension (PK) |

**Issue 1.4.A — No `gid` in profits**: Same as positions, no BIGSERIAL gid. Since `Profit` type has no gid db tag, this should not cause issues with StructScan.

---

### 1.5 `margin_loans` / `margin_repays` / `margin_interests` / `margin_liquidations`

All four tables match the original MySQL schema column-for-column (with TEXT replacing DECIMAL). Our Supabase adds `user_id` and `id UUID` for multi-tenancy. No missing columns.

**Verified**: margin_loans columns (exchange, transaction_id, asset, isolated_symbol, principle, time) match. margin_repays columns match. margin_interests columns (exchange, asset, isolated_symbol, principle, interest, interest_rate, time) match. margin_liquidations columns (exchange, symbol, side, order_id, price, quantity, average_price, executed_quantity, time_in_force, is_isolated, time) match.

---

### 1.6 `futures_position_risks`

| Column | MySQL (original) | Supabase (ours) | Status |
|--------|:---:|:---:|:---:|
| `gid` BIGINT AUTO_INCREMENT | YES | NO | MISSING |
| `exchange` VARCHAR(24) | YES | YES TEXT | OK |
| `symbol` VARCHAR(32) | YES | YES TEXT | OK |
| `position_side` VARCHAR(10) | YES | YES TEXT | OK |
| `leverage` DECIMAL(16,2) | YES | YES TEXT | OK |
| `liquidation_price` DECIMAL(16,8) | YES | YES TEXT | OK |
| `entry_price` DECIMAL(16,8) | YES | YES TEXT | OK |
| `mark_price` DECIMAL(16,8) | YES | YES TEXT | OK |
| `break_even_price` DECIMAL(16,8) | YES | YES TEXT | OK |
| `position_amount` DECIMAL(16,8) | YES | YES TEXT | OK |
| `unrealized_pnl` DECIMAL(16,2) | YES | YES TEXT | OK |
| `notional` DECIMAL(16,2) | YES | YES TEXT | OK |
| `initial_margin` DECIMAL(16,2) | YES | YES TEXT | OK |
| `maint_margin` DECIMAL(16,2) | YES | YES TEXT | OK |
| `position_initial_margin` DECIMAL(16,2) | YES | YES TEXT | OK |
| `open_order_initial_margin` DECIMAL(16,2) | YES | YES TEXT | OK |
| `adl` DECIMAL(16,2) | YES | YES TEXT | OK |
| `margin_asset` VARCHAR(20) | YES | YES TEXT | OK |
| `updated_at` DATETIME(3) | YES | YES TIMESTAMPTZ | OK |
| `strategy_instance_id` | NO | YES | SaaS extension |
| `user_id` | NO | YES | SaaS extension |

**Issue 1.6.A — No `gid` in futures_position_risks**: Missing auto-increment PK. Not critical since `PositionRisk` type does not have a gid tag.

**Issue 1.6.B — Unique key differences**: Original MySQL has `UNIQUE KEY (exchange, symbol, position_side, updated_at)` — a time-series unique key. Our migration 00036 DROPPED unique constraints entirely, making it append-only. This means we accumulate all position risk snapshots (no deduplication by time). This is a design choice: paper trading writes every 30s, so we want history.

---

### 1.7 `nav_history_details`

| Column | MySQL (original) | Supabase (ours) | Status |
|--------|:---:|:---:|:---:|
| `session` | YES | YES TEXT | OK |
| `exchange` | YES | YES TEXT | OK |
| `subaccount` | YES | YES TEXT | OK |
| `time` | YES | YES TIMESTAMPTZ | OK |
| `currency` | YES | YES TEXT | OK |
| `net_asset_in_usd` | YES | YES TEXT | OK |
| `net_asset_in_btc` | YES | YES TEXT | OK |
| `balance` | YES | YES TEXT | OK |
| `available` | YES | YES TEXT | OK |
| `locked` | YES | YES TEXT | OK |
| `borrowed` | YES | YES TEXT | OK |
| `net_asset` | YES | YES TEXT | OK |
| `price_in_usd` | YES | YES TEXT | OK |
| `is_margin` | YES | YES BOOLEAN | OK |
| `is_isolated` | YES | YES BOOLEAN | OK |
| `isolated_symbol` | YES | YES TEXT | OK |
| `interest` | NO (not in original INSERT) | YES TEXT | Our extension |

**Note**: Our Supabase nav_history_details has an extra `interest` TEXT column. Original MySQL schema does not have this. Not harmful.

---

### 1.8 Index Comparison

| Table | Original MySQL Unique Key | Our Supabase Unique Key | Match? |
|-------|:---:|:---:|:---:|
| `orders` | `(order_id, exchange)` | `(user_id, order_id, exchange)` | OK (multi-tenant) |
| `trades` | `(exchange, symbol, side, id)` | `(user_id, exchange, trade_id)` | DIFFERENT |
| `profits` | `(trade_id)` | `(user_id, exchange, symbol, side, trade_id)` | DIFFERENT |
| `positions` | `(trade_id, side, exchange)` | `(user_id, trade_id, side, symbol, exchange)` | OK (multi-tenant) |
| `futures_position_risks` | `(exchange, symbol, position_side, updated_at)` | NONE (dropped in 00036) | DIFFERENT |
| `margin_loans` | `(transaction_id)` | `(user_id, transaction_id)` | OK |
| `margin_repays` | `(transaction_id)` | `(user_id, transaction_id)` | OK |
| `margin_liquidations` | `(order_id, exchange)` | `(user_id, order_id, exchange)` | OK |

**Issue 1.8.A — trades unique key mismatch**: Original MySQL unique is `(exchange, symbol, side, id)` but ours is `(user_id, exchange, trade_id)`. The original includes `symbol` and `side` in the unique key — this means the same trade ID can exist on different symbols. Our key drops `symbol` and `side`, meaning a trade_id is globally unique per user+exchange. For paper trading, trade IDs are generated sequentially so this works, but for live trading with real exchange data, the same numeric trade ID could theoretically appear on different symbols.

**Issue 1.8.B — profits unique key**: Original is just `(trade_id)`, ours is `(user_id, exchange, symbol, side, trade_id)` — much more granular. The original design allows only one profit per trade_id globally. Our design allows one profit per trade within a specific exchange+symbol+side combination. This is arguably more correct for multi-symbol strategies.

---

## Part 2: Live Trading Logic Comparison

### 2.1 Code Path Alignment

The live trading path in our fork uses **exactly the same Go code** as the original bbgo for:
- Order submission through exchange adapters
- Trade collection via `TradeCollector`
- Position tracking via `Position` struct
- Profit calculation in strategies

The only difference is the **database layer**. Original bbgo uses MySQL/SQLite; our fork adds PostgreSQL/Supabase as a third backend.

### 2.2 Database Insert Logic

**Orders (live mode)**:
- Original MySQL: `INSERT ... ON DUPLICATE KEY UPDATE status, executed_quantity, is_working, updated_at`
- Our PostgreSQL: `INSERT ... ON CONFLICT (user_id, order_id, exchange) DO UPDATE SET status, executed_quantity, is_working, updated_at`
- **Compatible**: Same upsert logic, just PostgreSQL syntax with multi-tenant key.

**Trades (live mode)**:
- Original MySQL: `INSERT ... ON DUPLICATE KEY UPDATE` all fields
- Our PostgreSQL: `INSERT ... ON CONFLICT (user_id, exchange, trade_id) DO UPDATE SET` most fields
- **Compatible**: Same upsert behavior.

**Profits (live mode)**:
- Original MySQL: Plain `INSERT` (no conflict handling — relies on unique key `(trade_id)`)
- Our PostgreSQL: `INSERT ... ON CONFLICT DO NOTHING`
- **Compatible**: Original would error on duplicate; ours silently ignores. Same net effect since profits should not duplicate.

**Positions (live mode)**:
- Original MySQL: Plain `INSERT` (unique key `(trade_id, side, exchange)`)
- Our PostgreSQL: `INSERT ... ON CONFLICT DO NOTHING`
- **Compatible**: Same behavior.

### 2.3 Missing `created_at` column in trades

**Issue 2.3.A**: The original MySQL `trades` table does NOT have a `created_at` column — it uses `traded_at` for timestamping. The original SQLite migration `20240531163411_trades_created.sql` adds `created_at` to trades in SQLite only. Our Supabase trades table does not have `created_at` either. This is correct alignment with the MySQL schema.

### 2.4 `RecordPosition` — Critical Code Path

In the original bbgo, `TradeCollector` calls `RecordPosition` after processing a trade. This:
1. Updates the in-memory `Position` object (add/subtract base/quote)
2. Calculates profit when closing a position
3. Calls `PositionService.Insert()` to persist the position snapshot
4. Calls `ProfitService.Insert()` to persist the profit

**Issue 2.4.A**: Our fork's `PositionService.Insert` for postgres includes `net_profit` column, but the original MySQL `positions` DDL does not have `net_profit`. However, looking at the original `Insert` code, it passes `profit` and `netProfit` as separate parameters — and the SQL includes both `profit` and `net_profit` columns. This means the original MySQL migration must have added `net_profit` later. Checking... the original migration `20220307132917_add_positions.sql` does NOT include `net_profit` in the DDL. However, the Insert code passes both. This would fail on original MySQL too if `net_profit` column was not added later.

**UPDATE**: Checking the original more carefully — the original `PositionService.Insert` at `service/position.go:49-97` includes `profit` and `net_profit` in the INSERT but the MySQL DDL only has `profit`. This means either there is a missing migration in the original, or `net_profit` was added later. Our fork correctly has `net_profit` in the Supabase DDL, so this is actually **ahead of** the original.

---

## Part 3: Paper Trading Logic Comparison (Spot + Futures)

### 3.1 Spot Paper Trading

The paper trade engine (`pkg/bbgo/paper_trade_exchange.go`) implements `types.Exchange` to replace the real exchange. It:
1. Wraps a real exchange for market data
2. Maintains virtual balances
3. Matches orders against kline data
4. Generates trade callbacks with fees

**Comparison with live trading**:

| Aspect | Live (original bbgo) | Paper (our engine) | Aligned? |
|--------|:---:|:---:|:---:|
| Order submission | Sends to exchange API | Adds to local book | YES — different by design |
| Order matching | Exchange matches | Kline-driven matching | YES — different by design |
| Trade generation | Exchange reports fills | Generated on match | YES |
| Balance tracking | Exchange balance API | Local virtual balances | YES — different by design |
| Fee calculation | Exchange reports fee | 0.1% taker/maker fee | WARN — Simplified |
| Order types | LIMIT, MARKET, STOP_LIMIT, etc. | LIMIT, MARKET only | Issue |
| Balance locking | Exchange handles | `LockBalance`/`UnlockBalance` | YES |
| Open order management | Exchange order book | `paperMatchingBook` | YES |
| Fee currency | Exchange-specific (BNB discount etc) | Always quote currency | WARN — Simplified |

**Issue 3.1.A — Stop orders not supported**: The paper engine only handles LIMIT and MARKET orders. If a strategy submits STOP_LIMIT, STOP_MARKET, TAKE_PROFIT, or TAKE_PROFIT_MARKET orders, they are accepted but never triggered. This means strategies like `supertrend` that use stop orders would behave differently in paper vs live.

### 3.2 Futures Paper Trading

The futures paper engine (`pkg/bbgo/paper_trade_futures.go`) adds:
1. Short selling without base currency
2. Margin-based position tracking (notional/leverage)
3. Weighted average entry price
4. Unrealized PnL calculation
5. Liquidation price computation
6. Position risk sync to DB every 30s

**Comparison with live futures**:

| Aspect | Live Futures (original bbgo) | Paper Futures (our engine) | Aligned? |
|--------|:---:|:---:|:---:|
| Position opening | Exchange tracks | `paperFuturesState` tracking | YES |
| Entry price | Exchange reports | Weighted average | YES |
| Margin locked | Exchange manages | `notional / leverage` locked | YES |
| Unrealized PnL | Exchange `QueryPositionRisk` | Computed locally | YES |
| Liquidation price | Exchange reports | Computed: Long=entry*(1-1/lev+rate), Short=entry*(1+1/lev-rate) | WARN — Simplified |
| Leverage | User sets on exchange | From session config | YES |
| Maintenance margin rate | Exchange-specific tiers | Fixed 0.5% | WARN — Simplified |
| Position flip detection | Exchange handles | Long-to-Short/Short-to-Long detection | YES |
| Funding rate | Exchange charges | NOT IMPLEMENTED | Issue |
| ADL (Auto-Deleveraging) | Exchange computes | NOT IMPLEMENTED | Issue |
| Mark price | Exchange provides | Uses last kline close | WARN — Simplified |

**Issue 3.2.A — No funding rate**: Futures perpetual contracts have funding rate payments every 8 hours. The paper engine does not simulate this, so long-running futures positions will have incorrect PnL in paper mode vs live.

**Issue 3.2.B — Simplified liquidation**: Real exchanges use tiered maintenance margin rates based on position size. Our engine uses a flat 0.5% rate. Large positions would be liquidated at different prices in paper vs live.

**Issue 3.2.C — Mark price vs last price**: Real exchanges use a sophisticated mark price (index price + decaying basis). Our engine uses the last kline close as mark price, which can be more volatile.

### 3.3 Margin Paper Trading

| Aspect | Live Margin (original bbgo) | Paper Margin (our engine) | Aligned? |
|--------|:---:|:---:|:---:|
| Borrow asset | Exchange `BorrowMarginAsset` | Local balance addition | YES |
| Repay asset | Exchange `RepayMarginAsset` | Local balance deduction | YES |
| Interest accrual | Exchange charges | Hourly 0.01% rate | WARN — Simplified |
| Max borrowable | Exchange risk system | 5x available balance | WARN — Simplified |
| Liquidation | Exchange risk engine | NOT IMPLEMENTED | Issue |

**Issue 3.3.A — No margin liquidation**: The paper engine accrues interest but never liquidates positions that fall below maintenance margin. In real margin trading, exchange would force-liquidate.

**Issue 3.3.B — Simplified interest rate**: Real exchanges have variable interest rates per asset. Our engine uses a flat 0.01% hourly rate.

---

## Part 4: Other Supplementary Checks

### 4.1 SyncTask and SyncService

The original bbgo's `SyncService` syncs historical orders/trades from the exchange on startup. Our fork adds `TablePrefix` and `UserID` to the service, with postgres-specific SQL.

**Issue 4.1.A — `SyncTask.execute` uses table name directly**: The `SyncTask` struct references table names hardcoded (e.g., `"orders"`, `"trades"`). With `TablePrefix`, the `SelectLastOrders`/`SelectLastTrades` builders still use hardcoded table names. When `TablePrefix` is `paper_`, the sync would target the wrong table. However, paper mode does not sync from exchanges (there is no exchange to sync from), so this is not a practical issue.

### 4.2 TradeCollector

The `TradeCollector` is the core component that processes trades and triggers position/profit recording. Both codebases use the same implementation. The key method `RecordTrade`:
1. Receives a new trade from exchange
2. Updates the associated order
3. Calls `Position.AddTrade(trade, ...)`
4. If position is closed, calculates profit and calls `ProfitService.Insert()`
5. Calls `PositionService.Insert()` to record the position snapshot
6. Emits callbacks

**Aligned**: The trade collection logic is identical between our fork and the original.

### 4.3 Strategy Execution

Strategies in both codebases:
1. Receive market data via WebSocket or gRPC
2. Make trading decisions
3. Submit orders via `OrderExecutor`
4. Receive callbacks for fills/trades

The `OrderExecutor` interface is the same. In live mode, orders go to the real exchange. In paper mode, orders go to `PaperTradeExchange`.

**Aligned**: Strategy execution logic is unchanged.

### 4.4 Paper Balances Persistence

**Issue 4.4.A — Paper balances in Supabase**: Migration `00027_paper_balances.sql` creates a `paper_balances` table. However, the paper trade engine stores balances in memory (`PaperTradeExchange.balances` map). There is no code to persist paper balances to the `paper_balances` table on each change. The table exists but may not be populated.

### 4.5 Order Restoration on Restart

The original bbgo restores open orders from the database on restart. For paper trading, `PaperTradeExchange.RestoreOpenOrdersFromDB` queries orders from the Supabase `paper_orders` table.

**Issue 4.5.A — Paper futures position restoration**: When a paper container restarts, open orders are restored from DB, but **futures positions are NOT restored**. The `paperFuturesState` starts empty. This means a running short position from before the restart would be lost. The paper engine would treat the next trade as opening a new position rather than continuing the existing one.

### 4.6 Column Type Mismatches (TEXT vs fixedpoint.Value)

All numeric columns in Supabase use `TEXT` type. The Go `fixedpoint.Value` marshals to string for inserts and unmarshals from string on reads. This works correctly with PostgreSQL since:
- `fixedpoint.Value` implements `sql.Scanner` and `driver.Valuer`
- TEXT columns accept string values
- `strconv.ParseFloat` is not involved (avoids precision loss)

**Aligned**: This is a deliberate design choice, not a bug.

### 4.7 `created_at` Timestamp Handling

**Issue 4.7.A — `created_at` defaults differ**: In original MySQL, `orders.created_at` is set by the INSERT statement using `:created_at` (from `order.CreationTime`). In our PostgreSQL, it is also set via parameter. But the Supabase DDL has `DEFAULT now()` as a fallback. This means if `CreationTime` is zero, it would insert the zero time rather than `now()`. Original MySQL has the same behavior. **Aligned**.

### 4.8 Missing `ReduceOnly` and `ClosePosition` in Schema

The `SubmitOrder` struct has `ReduceOnly` and `ClosePosition` fields with `db:"reduce_only"` and `db:"close_position"` tags. These are **not in the Supabase schema** for orders. They are also not in the original MySQL `orders` table DDL. These fields are used only in the exchange API submission, not persisted to the database. **Aligned with original**.

### 4.9 Supabase Type Generation

The `pnpm sb go-types` command regenerates `manager/supabase_types.go` and `pkg/supabasetypes/database_types.go`. These types should reflect the current Supabase schema. If migrations are applied but types are not regenerated, the Go types would be stale.

**Issue 4.9.A**: The auto-generated types use string fields for TEXT columns and boolean for BOOLEAN columns, matching the Supabase DDL. However, the service layer uses `fixedpoint.Value` and `types.Order` for inserts — these Go types use `fixedpoint.Value` which serializes to string. The type generation produces separate types that may not be directly compatible with the service layer types. This is fine as long as they are used for different purposes (API responses vs database operations).

---

## Summary of Actionable Issues

### Critical (must fix)

1. **Issue 1.2.A — `trade_id` column name mismatch** ✅ FIXED: Added `postgresTradeSelectColumns` with `trade_id AS id` alias in `genTradeSelectColumns`. Updated `SelectLastTrades`, `detectLastestSelfTrade`, and all postgres SELECT paths.

2. **Issue 1.1.A — `order_uuid` column name mismatch** ✅ FIXED: Added `order_uuid AS uuid` alias in `genOrderSQL` and `SelectLastOrders` for postgres driver.

3. **Issue 3.2.A — No funding rate in paper futures** ✅ FIXED: Added `applyFundingRate()` with 0.01% rate every 8h. Longs pay shorts (positive rate). Wired into `StartBackgroundServices`. Balance updates logged.

4. **Issue 4.5.A — Paper futures positions not restored on restart** ✅ FIXED: Added `restoreFuturesPositions()` method that queries latest `futures_position_risks` snapshots with `DISTINCT ON (symbol)` and reconstructs `paperFuturesState`. Called from `RestoreFromDB` when futures mode is enabled.

### High (should fix)

5. **Issue 3.1.A — Stop orders not supported in paper** ✅ FIXED: Added `stopOrders` slice to `paperMatchingBook`. `SubmitOrder` routes stop/take-profit orders to stop book. `checkStopTriggers()` in `ProcessKLine` triggers when kline high/low crosses stop price. `CancelOrder` handles stop order cancellation with proper balance unlock.

6. **Issue 1.8.A — Trades unique key missing `symbol` and `side`** ✅ FIXED: Migration `00037_trades_unique_add_symbol_side.sql` changes unique to `(user_id, exchange, symbol, side, trade_id)`. Updated `ON CONFLICT` clause in trade INSERT.

7. **Issue 3.2.B — Simplified liquidation** ✅ FIXED: Replaced flat 0.5% with Binance-style tiered rates (`getMaintenanceMarginRate`): 0.4% up to 50K, 0.5% to 250K, 1% to 1M, 2.5% to 5M, 5% to 10M, 10% above.

### Medium (nice to have)

8. **Issue 3.2.C — Mark price simplification**: Using last kline close instead of proper mark price. **Remaining** — requires index price feed which isn't available in paper mode.

9. **Issue 3.3.A — No margin liquidation in paper**: Paper margin positions can go indefinitely negative. **Remaining** — low priority since margin mode is rarely used.

10. **Issue 4.4.A — Paper balances table not populated** ✅ VERIFIED WORKING: `syncBalances()` is already wired via `BindUserData` callback. Every balance change triggers `upsertBalances()` which writes to `paper_balances` with `ON CONFLICT (user_id, currency)`.

### Informational

11. **SaaS extensions** (`user_id`, `strategy_instance_id`, `position_action`): These are intentional multi-tenant additions, not bugs.

12. **TEXT vs DECIMAL**: Deliberate design choice for arbitrary precision.

13. **No `gid` in positions/profits**: Not needed since these types do not reference gid for ordering in the SaaS context.
