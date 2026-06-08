package main

type PublicUserProfilesSelect struct {
  AvatarUrl   *string `json:"avatar_url,omitempty"`
  CreatedAt   string  `json:"created_at"`
  DisplayName *string `json:"display_name,omitempty"`
  Email       string  `json:"email"`
  Id          string  `json:"id"`
  Role        string  `json:"role"`
  UpdatedAt   string  `json:"updated_at"`
}

type PublicUserProfilesInsert struct {
  AvatarUrl   *string `json:"avatar_url,omitempty"`
  CreatedAt   *string `json:"created_at,omitempty"`
  DisplayName *string `json:"display_name,omitempty"`
  Email       string  `json:"email"`
  Id          string  `json:"id"`
  Role        *string `json:"role,omitempty"`
  UpdatedAt   *string `json:"updated_at,omitempty"`
}

type PublicUserProfilesUpdate struct {
  AvatarUrl   *string `json:"avatar_url,omitempty"`
  CreatedAt   *string `json:"created_at,omitempty"`
  DisplayName *string `json:"display_name,omitempty"`
  Email       *string `json:"email,omitempty"`
  Id          *string `json:"id,omitempty,omitempty"`
  Role        *string `json:"role,omitempty"`
  UpdatedAt   *string `json:"updated_at,omitempty"`
}

type PublicExchangeCredentialsSelect struct {
  ApiKeyEncrypted     string  `json:"api_key_encrypted"`
  ApiSecretEncrypted  string  `json:"api_secret_encrypted"`
  CreatedAt           string  `json:"created_at"`
  Exchange            string  `json:"exchange"`
  Id                  string  `json:"id"`
  IsTestnet           bool    `json:"is_testnet"`
  IsVerified          bool    `json:"is_verified"`
  LastVerifiedAt      *string `json:"last_verified_at,omitempty"`
  PassphraseEncrypted *string `json:"passphrase_encrypted,omitempty"`
  UserId              string  `json:"user_id"`
}

type PublicExchangeCredentialsInsert struct {
  ApiKeyEncrypted     string  `json:"api_key_encrypted"`
  ApiSecretEncrypted  string  `json:"api_secret_encrypted"`
  CreatedAt           *string `json:"created_at,omitempty"`
  Exchange            string  `json:"exchange"`
  Id                  *string `json:"id,omitempty,omitempty"`
  IsTestnet           *bool `json:"is_testnet,omitempty"`
  IsVerified          *bool `json:"is_verified,omitempty"`
  LastVerifiedAt      *string `json:"last_verified_at,omitempty"`
  PassphraseEncrypted *string `json:"passphrase_encrypted,omitempty"`
  UserId              string  `json:"user_id"`
}

type PublicExchangeCredentialsUpdate struct {
  ApiKeyEncrypted     *string `json:"api_key_encrypted,omitempty"`
  ApiSecretEncrypted  *string `json:"api_secret_encrypted,omitempty"`
  CreatedAt           *string `json:"created_at,omitempty"`
  Exchange            *string `json:"exchange,omitempty"`
  Id                  *string `json:"id,omitempty,omitempty"`
  IsTestnet           *bool `json:"is_testnet,omitempty"`
  IsVerified          *bool `json:"is_verified,omitempty"`
  LastVerifiedAt      *string `json:"last_verified_at,omitempty"`
  PassphraseEncrypted *string `json:"passphrase_encrypted,omitempty"`
  UserId              *string `json:"user_id,omitempty"`
}

type PublicOrdersSelect struct {
  ActualOrderId      int64   `json:"actual_order_id"`
  ClientOrderId      string  `json:"client_order_id"`
  CreatedAt          string  `json:"created_at"`
  Exchange           string  `json:"exchange"`
  ExecutedQuantity   *string `json:"executed_quantity,omitempty"`
  Id                 string  `json:"id"`
  IsFutures          bool    `json:"is_futures"`
  IsIsolated         bool    `json:"is_isolated"`
  IsMargin           bool    `json:"is_margin"`
  IsWorking          bool    `json:"is_working"`
  OrderId            string  `json:"order_id"`
  OrderType          string  `json:"order_type"`
  OrderUuid          string  `json:"order_uuid"`
  Price              string  `json:"price"`
  Quantity           string  `json:"quantity"`
  Side               string  `json:"side"`
  Status             string  `json:"status"`
  StopPrice          string  `json:"stop_price"`
  StrategyInstanceId string  `json:"strategy_instance_id"`
  Symbol             string  `json:"symbol"`
  TimeInForce        string  `json:"time_in_force"`
  UpdatedAt          string  `json:"updated_at"`
  UserId             string  `json:"user_id"`
}

type PublicOrdersInsert struct {
  ActualOrderId      *int64  `json:"actual_order_id"`
  ClientOrderId      *string `json:"client_order_id,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  ExecutedQuantity   *string `json:"executed_quantity,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  IsWorking          *bool `json:"is_working,omitempty"`
  OrderId            string  `json:"order_id"`
  OrderType          string  `json:"order_type"`
  OrderUuid          *string `json:"order_uuid,omitempty"`
  Price              string  `json:"price"`
  Quantity           string  `json:"quantity"`
  Side               string  `json:"side"`
  Status             string  `json:"status"`
  StopPrice          *string `json:"stop_price,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             string  `json:"symbol"`
  TimeInForce        *string `json:"time_in_force,omitempty"`
  UpdatedAt          *string `json:"updated_at,omitempty"`
  UserId             string  `json:"user_id"`
}

type PublicOrdersUpdate struct {
  ActualOrderId      *int64  `json:"actual_order_id"`
  ClientOrderId      *string `json:"client_order_id,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  ExecutedQuantity   *string `json:"executed_quantity,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  IsWorking          *bool `json:"is_working,omitempty"`
  OrderId            *string `json:"order_id,omitempty"`
  OrderType          *string `json:"order_type,omitempty"`
  OrderUuid          *string `json:"order_uuid,omitempty"`
  Price              *string `json:"price,omitempty"`
  Quantity           *string `json:"quantity,omitempty"`
  Side               *string `json:"side,omitempty"`
  Status             *string `json:"status,omitempty"`
  StopPrice          *string `json:"stop_price,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TimeInForce        *string `json:"time_in_force,omitempty"`
  UpdatedAt          *string `json:"updated_at,omitempty"`
  UserId             *string `json:"user_id,omitempty"`
}

type PublicTradesSelect struct {
  Exchange           string  `json:"exchange"`
  Fee                string  `json:"fee"`
  FeeCurrency        string  `json:"fee_currency"`
  Id                 string  `json:"id"`
  IsBuyer            bool    `json:"is_buyer"`
  IsFutures          bool    `json:"is_futures"`
  IsIsolated         bool    `json:"is_isolated"`
  IsMaker            bool    `json:"is_maker"`
  IsMargin           bool    `json:"is_margin"`
  OrderId            string  `json:"order_id"`
  OrderUuid          string  `json:"order_uuid"`
  Pnl                *string `json:"pnl,omitempty"`
  Price              string  `json:"price"`
  Quantity           string  `json:"quantity"`
  QuoteQuantity      *string `json:"quote_quantity,omitempty"`
  Side               string  `json:"side"`
  Strategy           string  `json:"strategy"`
  StrategyInstanceId string  `json:"strategy_instance_id"`
  Symbol             string  `json:"symbol"`
  TradeId            string  `json:"trade_id"`
  TradedAt           *string `json:"traded_at,omitempty"`
  UserId             string  `json:"user_id"`
}

type PublicTradesInsert struct {
  Exchange           *string `json:"exchange,omitempty"`
  Fee                string  `json:"fee"`
  FeeCurrency        string  `json:"fee_currency"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsBuyer            *bool `json:"is_buyer,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMaker            *bool `json:"is_maker,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  OrderId            string  `json:"order_id"`
  OrderUuid          *string `json:"order_uuid,omitempty"`
  Pnl                *string `json:"pnl,omitempty"`
  Price              string  `json:"price"`
  Quantity           string  `json:"quantity"`
  QuoteQuantity      *string `json:"quote_quantity,omitempty"`
  Side               string  `json:"side"`
  Strategy           *string `json:"strategy,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             string  `json:"symbol"`
  TradeId            string  `json:"trade_id"`
  TradedAt           *string `json:"traded_at,omitempty"`
  UserId             string  `json:"user_id"`
}

type PublicTradesUpdate struct {
  Exchange           *string `json:"exchange,omitempty"`
  Fee                *string `json:"fee,omitempty"`
  FeeCurrency        *string `json:"fee_currency,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsBuyer            *bool `json:"is_buyer,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMaker            *bool `json:"is_maker,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  OrderId            *string `json:"order_id,omitempty"`
  OrderUuid          *string `json:"order_uuid,omitempty"`
  Pnl                *string `json:"pnl,omitempty"`
  Price              *string `json:"price,omitempty"`
  Quantity           *string `json:"quantity,omitempty"`
  QuoteQuantity      *string `json:"quote_quantity,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           *string `json:"strategy,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TradeId            *string `json:"trade_id,omitempty"`
  TradedAt           *string `json:"traded_at,omitempty"`
  UserId             *string `json:"user_id,omitempty"`
}

type PublicBacktestReportsSelect struct {
  Cagr         *string `json:"cagr,omitempty"`
  Config       interface{} `json:"config"`
  CreatedAt    string      `json:"created_at"`
  EndDate      string      `json:"end_date"`
  Id           string      `json:"id"`
  LossCount    int32       `json:"loss_count"`
  MaxDrawdown  string      `json:"max_drawdown"`
  ProfitFactor *string `json:"profit_factor,omitempty"`
  ReportJson   interface{} `json:"report_json"`
  SharpeRatio  *string `json:"sharpe_ratio,omitempty"`
  SortinoRatio *string `json:"sortino_ratio,omitempty"`
  StartDate    string      `json:"start_date"`
  Strategy     string      `json:"strategy"`
  TotalProfit  string      `json:"total_profit"`
  TotalTrades  int32       `json:"total_trades"`
  UserId       string      `json:"user_id"`
  WinCount     int32       `json:"win_count"`
  WinRate      string      `json:"win_rate"`
}

type PublicBacktestReportsInsert struct {
  Cagr         *string `json:"cagr,omitempty"`
  Config       interface{} `json:"config"`
  CreatedAt    *string `json:"created_at,omitempty"`
  EndDate      string      `json:"end_date"`
  Id           *string `json:"id,omitempty,omitempty"`
  LossCount    int32       `json:"loss_count"`
  MaxDrawdown  string      `json:"max_drawdown"`
  ProfitFactor *string `json:"profit_factor,omitempty"`
  ReportJson   interface{} `json:"report_json"`
  SharpeRatio  *string `json:"sharpe_ratio,omitempty"`
  SortinoRatio *string `json:"sortino_ratio,omitempty"`
  StartDate    string      `json:"start_date"`
  Strategy     string      `json:"strategy"`
  TotalProfit  string      `json:"total_profit"`
  TotalTrades  int32       `json:"total_trades"`
  UserId       string      `json:"user_id"`
  WinCount     int32       `json:"win_count"`
  WinRate      string      `json:"win_rate"`
}

type PublicBacktestReportsUpdate struct {
  Cagr         *string `json:"cagr,omitempty"`
  Config       interface{} `json:"config"`
  CreatedAt    *string `json:"created_at,omitempty"`
  EndDate      *string `json:"end_date,omitempty"`
  Id           *string `json:"id,omitempty,omitempty"`
  LossCount    *int32      `json:"loss_count"`
  MaxDrawdown  *string `json:"max_drawdown,omitempty"`
  ProfitFactor *string `json:"profit_factor,omitempty"`
  ReportJson   interface{} `json:"report_json"`
  SharpeRatio  *string `json:"sharpe_ratio,omitempty"`
  SortinoRatio *string `json:"sortino_ratio,omitempty"`
  StartDate    *string `json:"start_date,omitempty"`
  Strategy     *string `json:"strategy,omitempty"`
  TotalProfit  *string `json:"total_profit,omitempty"`
  TotalTrades  *int32      `json:"total_trades"`
  UserId       *string `json:"user_id,omitempty"`
  WinCount     *int32      `json:"win_count"`
  WinRate      *string `json:"win_rate,omitempty"`
}

type PublicPositionsSelect struct {
  AverageCost        string  `json:"average_cost"`
  Base               string  `json:"base"`
  BaseCurrency       string  `json:"base_currency"`
  CreatedAt          string  `json:"created_at"`
  Exchange           string  `json:"exchange"`
  Id                 string  `json:"id"`
  NetProfit          *string `json:"net_profit,omitempty"`
  Profit             *string `json:"profit,omitempty"`
  Quote              string  `json:"quote"`
  QuoteCurrency      string  `json:"quote_currency"`
  Side               string  `json:"side"`
  Strategy           string  `json:"strategy"`
  StrategyInstanceId string  `json:"strategy_instance_id"`
  Symbol             string  `json:"symbol"`
  TradeId            int64   `json:"trade_id"`
  TradedAt           string  `json:"traded_at"`
  UserId             string  `json:"user_id"`
}

type PublicPositionsInsert struct {
  AverageCost        *string `json:"average_cost,omitempty"`
  Base               *string `json:"base,omitempty"`
  BaseCurrency       *string `json:"base_currency,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  NetProfit          *string `json:"net_profit,omitempty"`
  Profit             *string `json:"profit,omitempty"`
  Quote              *string `json:"quote,omitempty"`
  QuoteCurrency      *string `json:"quote_currency,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           string  `json:"strategy"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             string  `json:"symbol"`
  TradeId            int64   `json:"trade_id"`
  TradedAt           string  `json:"traded_at"`
  UserId             string  `json:"user_id"`
}

type PublicPositionsUpdate struct {
  AverageCost        *string `json:"average_cost,omitempty"`
  Base               *string `json:"base,omitempty"`
  BaseCurrency       *string `json:"base_currency,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  NetProfit          *string `json:"net_profit,omitempty"`
  Profit             *string `json:"profit,omitempty"`
  Quote              *string `json:"quote,omitempty"`
  QuoteCurrency      *string `json:"quote_currency,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           *string `json:"strategy,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TradeId            *int64  `json:"trade_id"`
  TradedAt           *string `json:"traded_at,omitempty"`
  UserId             *string `json:"user_id,omitempty"`
}

type PublicProfitsSelect struct {
  AverageCost        string  `json:"average_cost"`
  BaseCurrency       string  `json:"base_currency"`
  CreatedAt          string  `json:"created_at"`
  Exchange           string  `json:"exchange"`
  Fee                string  `json:"fee"`
  FeeCurrency        string  `json:"fee_currency"`
  FeeInUsd           *string `json:"fee_in_usd,omitempty"`
  Id                 string  `json:"id"`
  IsBuyer            bool    `json:"is_buyer"`
  IsFutures          bool    `json:"is_futures"`
  IsIsolated         bool    `json:"is_isolated"`
  IsMaker            bool    `json:"is_maker"`
  IsMargin           bool    `json:"is_margin"`
  NetProfit          string  `json:"net_profit"`
  NetProfitMargin    string  `json:"net_profit_margin"`
  Price              string  `json:"price"`
  Profit             string  `json:"profit"`
  ProfitMargin       string  `json:"profit_margin"`
  Quantity           string  `json:"quantity"`
  QuoteCurrency      string  `json:"quote_currency"`
  QuoteQuantity      string  `json:"quote_quantity"`
  Side               string  `json:"side"`
  Strategy           string  `json:"strategy"`
  StrategyInstanceId string  `json:"strategy_instance_id"`
  Symbol             string  `json:"symbol"`
  TradeId            int64   `json:"trade_id"`
  TradedAt           string  `json:"traded_at"`
  UserId             string  `json:"user_id"`
}

type PublicProfitsInsert struct {
  AverageCost        *string `json:"average_cost,omitempty"`
  BaseCurrency       *string `json:"base_currency,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  Fee                *string `json:"fee,omitempty"`
  FeeCurrency        *string `json:"fee_currency,omitempty"`
  FeeInUsd           *string `json:"fee_in_usd,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsBuyer            *bool `json:"is_buyer,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMaker            *bool `json:"is_maker,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  NetProfit          *string `json:"net_profit,omitempty"`
  NetProfitMargin    *string `json:"net_profit_margin,omitempty"`
  Price              *string `json:"price,omitempty"`
  Profit             *string `json:"profit,omitempty"`
  ProfitMargin       *string `json:"profit_margin,omitempty"`
  Quantity           *string `json:"quantity,omitempty"`
  QuoteCurrency      *string `json:"quote_currency,omitempty"`
  QuoteQuantity      *string `json:"quote_quantity,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           string  `json:"strategy"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             string  `json:"symbol"`
  TradeId            int64   `json:"trade_id"`
  TradedAt           string  `json:"traded_at"`
  UserId             string  `json:"user_id"`
}

type PublicProfitsUpdate struct {
  AverageCost        *string `json:"average_cost,omitempty"`
  BaseCurrency       *string `json:"base_currency,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  Fee                *string `json:"fee,omitempty"`
  FeeCurrency        *string `json:"fee_currency,omitempty"`
  FeeInUsd           *string `json:"fee_in_usd,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsBuyer            *bool `json:"is_buyer,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMaker            *bool `json:"is_maker,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  NetProfit          *string `json:"net_profit,omitempty"`
  NetProfitMargin    *string `json:"net_profit_margin,omitempty"`
  Price              *string `json:"price,omitempty"`
  Profit             *string `json:"profit,omitempty"`
  ProfitMargin       *string `json:"profit_margin,omitempty"`
  Quantity           *string `json:"quantity,omitempty"`
  QuoteCurrency      *string `json:"quote_currency,omitempty"`
  QuoteQuantity      *string `json:"quote_quantity,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           *string `json:"strategy,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TradeId            *int64  `json:"trade_id"`
  TradedAt           *string `json:"traded_at,omitempty"`
  UserId             *string `json:"user_id,omitempty"`
}

type PublicStrategyRegistrySelect struct {
  Category        string      `json:"category"`
  CreatedAt       *string `json:"created_at,omitempty"`
  CrossExchange   *bool `json:"cross_exchange,omitempty"`
  Defaults        interface{} `json:"defaults"`
  Description     *string `json:"description,omitempty"`
  DisplayName     string      `json:"display_name"`
  Enabled         *bool `json:"enabled,omitempty"`
  Exchanges       interface{} `json:"exchanges"`
  Fields          interface{} `json:"fields"`
  Id              string      `json:"id"`
  LiveOnly        *bool `json:"live_only,omitempty"`
  RequiresFutures *bool `json:"requires_futures,omitempty"`
  SessionRoles    interface{} `json:"session_roles"`
  SortOrder       *int32      `json:"sort_order"`
  UpdatedAt       *string `json:"updated_at,omitempty"`
}

type PublicStrategyRegistryInsert struct {
  Category        *string `json:"category,omitempty"`
  CreatedAt       *string `json:"created_at,omitempty"`
  CrossExchange   *bool `json:"cross_exchange,omitempty"`
  Defaults        interface{} `json:"defaults"`
  Description     *string `json:"description,omitempty"`
  DisplayName     string      `json:"display_name"`
  Enabled         *bool `json:"enabled,omitempty"`
  Exchanges       interface{} `json:"exchanges"`
  Fields          interface{} `json:"fields"`
  Id              string      `json:"id"`
  LiveOnly        *bool `json:"live_only,omitempty"`
  RequiresFutures *bool `json:"requires_futures,omitempty"`
  SessionRoles    interface{} `json:"session_roles"`
  SortOrder       *int32      `json:"sort_order"`
  UpdatedAt       *string `json:"updated_at,omitempty"`
}

type PublicStrategyRegistryUpdate struct {
  Category        *string `json:"category,omitempty"`
  CreatedAt       *string `json:"created_at,omitempty"`
  CrossExchange   *bool `json:"cross_exchange,omitempty"`
  Defaults        interface{} `json:"defaults"`
  Description     *string `json:"description,omitempty"`
  DisplayName     *string `json:"display_name,omitempty"`
  Enabled         *bool `json:"enabled,omitempty"`
  Exchanges       interface{} `json:"exchanges"`
  Fields          interface{} `json:"fields"`
  Id              *string `json:"id,omitempty,omitempty"`
  LiveOnly        *bool `json:"live_only,omitempty"`
  RequiresFutures *bool `json:"requires_futures,omitempty"`
  SessionRoles    interface{} `json:"session_roles"`
  SortOrder       *int32      `json:"sort_order"`
  UpdatedAt       *string `json:"updated_at,omitempty"`
}

type PublicStrategyInstancesSelect struct {
  Config        interface{} `json:"config"`
  CreatedAt     string      `json:"created_at"`
  CrossExchange bool        `json:"cross_exchange"`
  Exchange      string      `json:"exchange"`
  InstanceId    string      `json:"instance_id"`
  Mode          string      `json:"mode"`
  Name          string      `json:"name"`
  Sessions      interface{} `json:"sessions"`
  Strategy      string      `json:"strategy"`
  Symbol        string      `json:"symbol"`
  UpdatedAt     string      `json:"updated_at"`
  UserId        string      `json:"user_id"`
}

type PublicStrategyInstancesInsert struct {
  Config        interface{} `json:"config"`
  CreatedAt     *string `json:"created_at,omitempty"`
  CrossExchange *bool `json:"cross_exchange,omitempty"`
  Exchange      *string `json:"exchange,omitempty"`
  InstanceId    string      `json:"instance_id"`
  Mode          string      `json:"mode"`
  Name          *string `json:"name,omitempty"`
  Sessions      interface{} `json:"sessions"`
  Strategy      string      `json:"strategy"`
  Symbol        *string `json:"symbol,omitempty"`
  UpdatedAt     *string `json:"updated_at,omitempty"`
  UserId        string      `json:"user_id"`
}

type PublicStrategyInstancesUpdate struct {
  Config        interface{} `json:"config"`
  CreatedAt     *string `json:"created_at,omitempty"`
  CrossExchange *bool `json:"cross_exchange,omitempty"`
  Exchange      *string `json:"exchange,omitempty"`
  InstanceId    *string `json:"instance_id,omitempty"`
  Mode          *string `json:"mode,omitempty"`
  Name          *string `json:"name,omitempty"`
  Sessions      interface{} `json:"sessions"`
  Strategy      *string `json:"strategy,omitempty"`
  Symbol        *string `json:"symbol,omitempty"`
  UpdatedAt     *string `json:"updated_at,omitempty"`
  UserId        *string `json:"user_id,omitempty"`
}

type PublicPaperOrdersSelect struct {
  ActualOrderId      int64   `json:"actual_order_id"`
  ClientOrderId      string  `json:"client_order_id"`
  CreatedAt          string  `json:"created_at"`
  Exchange           string  `json:"exchange"`
  ExecutedQuantity   *string `json:"executed_quantity,omitempty"`
  Id                 string  `json:"id"`
  IsFutures          bool    `json:"is_futures"`
  IsIsolated         bool    `json:"is_isolated"`
  IsMargin           bool    `json:"is_margin"`
  IsWorking          bool    `json:"is_working"`
  OrderId            string  `json:"order_id"`
  OrderType          string  `json:"order_type"`
  OrderUuid          string  `json:"order_uuid"`
  Price              string  `json:"price"`
  Quantity           string  `json:"quantity"`
  Side               string  `json:"side"`
  Status             string  `json:"status"`
  StopPrice          string  `json:"stop_price"`
  StrategyInstanceId string  `json:"strategy_instance_id"`
  Symbol             string  `json:"symbol"`
  TimeInForce        string  `json:"time_in_force"`
  UpdatedAt          string  `json:"updated_at"`
  UserId             string  `json:"user_id"`
}

type PublicPaperOrdersInsert struct {
  ActualOrderId      *int64  `json:"actual_order_id"`
  ClientOrderId      *string `json:"client_order_id,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  ExecutedQuantity   *string `json:"executed_quantity,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  IsWorking          *bool `json:"is_working,omitempty"`
  OrderId            *string `json:"order_id,omitempty"`
  OrderType          *string `json:"order_type,omitempty"`
  OrderUuid          *string `json:"order_uuid,omitempty"`
  Price              *string `json:"price,omitempty"`
  Quantity           *string `json:"quantity,omitempty"`
  Side               *string `json:"side,omitempty"`
  Status             *string `json:"status,omitempty"`
  StopPrice          *string `json:"stop_price,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TimeInForce        *string `json:"time_in_force,omitempty"`
  UpdatedAt          *string `json:"updated_at,omitempty"`
  UserId             string  `json:"user_id"`
}

type PublicPaperOrdersUpdate struct {
  ActualOrderId      *int64  `json:"actual_order_id"`
  ClientOrderId      *string `json:"client_order_id,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  ExecutedQuantity   *string `json:"executed_quantity,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  IsWorking          *bool `json:"is_working,omitempty"`
  OrderId            *string `json:"order_id,omitempty"`
  OrderType          *string `json:"order_type,omitempty"`
  OrderUuid          *string `json:"order_uuid,omitempty"`
  Price              *string `json:"price,omitempty"`
  Quantity           *string `json:"quantity,omitempty"`
  Side               *string `json:"side,omitempty"`
  Status             *string `json:"status,omitempty"`
  StopPrice          *string `json:"stop_price,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TimeInForce        *string `json:"time_in_force,omitempty"`
  UpdatedAt          *string `json:"updated_at,omitempty"`
  UserId             *string `json:"user_id,omitempty"`
}

type PublicPaperTradesSelect struct {
  Exchange           string  `json:"exchange"`
  Fee                string  `json:"fee"`
  FeeCurrency        string  `json:"fee_currency"`
  Id                 string  `json:"id"`
  IsBuyer            bool    `json:"is_buyer"`
  IsFutures          bool    `json:"is_futures"`
  IsIsolated         bool    `json:"is_isolated"`
  IsMaker            bool    `json:"is_maker"`
  IsMargin           bool    `json:"is_margin"`
  OrderId            string  `json:"order_id"`
  OrderUuid          string  `json:"order_uuid"`
  Pnl                *string `json:"pnl,omitempty"`
  Price              string  `json:"price"`
  Quantity           string  `json:"quantity"`
  QuoteQuantity      *string `json:"quote_quantity,omitempty"`
  Side               string  `json:"side"`
  Strategy           string  `json:"strategy"`
  StrategyInstanceId string  `json:"strategy_instance_id"`
  Symbol             string  `json:"symbol"`
  TradeId            string  `json:"trade_id"`
  TradedAt           *string `json:"traded_at,omitempty"`
  UserId             string  `json:"user_id"`
}

type PublicPaperTradesInsert struct {
  Exchange           *string `json:"exchange,omitempty"`
  Fee                *string `json:"fee,omitempty"`
  FeeCurrency        *string `json:"fee_currency,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsBuyer            *bool `json:"is_buyer,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMaker            *bool `json:"is_maker,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  OrderId            *string `json:"order_id,omitempty"`
  OrderUuid          *string `json:"order_uuid,omitempty"`
  Pnl                *string `json:"pnl,omitempty"`
  Price              *string `json:"price,omitempty"`
  Quantity           *string `json:"quantity,omitempty"`
  QuoteQuantity      *string `json:"quote_quantity,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           *string `json:"strategy,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TradeId            *string `json:"trade_id,omitempty"`
  TradedAt           *string `json:"traded_at,omitempty"`
  UserId             string  `json:"user_id"`
}

type PublicPaperTradesUpdate struct {
  Exchange           *string `json:"exchange,omitempty"`
  Fee                *string `json:"fee,omitempty"`
  FeeCurrency        *string `json:"fee_currency,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsBuyer            *bool `json:"is_buyer,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMaker            *bool `json:"is_maker,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  OrderId            *string `json:"order_id,omitempty"`
  OrderUuid          *string `json:"order_uuid,omitempty"`
  Pnl                *string `json:"pnl,omitempty"`
  Price              *string `json:"price,omitempty"`
  Quantity           *string `json:"quantity,omitempty"`
  QuoteQuantity      *string `json:"quote_quantity,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           *string `json:"strategy,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TradeId            *string `json:"trade_id,omitempty"`
  TradedAt           *string `json:"traded_at,omitempty"`
  UserId             *string `json:"user_id,omitempty"`
}

type PublicPaperPositionsSelect struct {
  AverageCost        string  `json:"average_cost"`
  Base               string  `json:"base"`
  BaseCurrency       string  `json:"base_currency"`
  CreatedAt          string  `json:"created_at"`
  Exchange           string  `json:"exchange"`
  Id                 string  `json:"id"`
  NetProfit          *string `json:"net_profit,omitempty"`
  Profit             *string `json:"profit,omitempty"`
  Quote              string  `json:"quote"`
  QuoteCurrency      string  `json:"quote_currency"`
  Side               string  `json:"side"`
  Strategy           string  `json:"strategy"`
  StrategyInstanceId string  `json:"strategy_instance_id"`
  Symbol             string  `json:"symbol"`
  TradeId            int64   `json:"trade_id"`
  TradedAt           string  `json:"traded_at"`
  UserId             string  `json:"user_id"`
}

type PublicPaperPositionsInsert struct {
  AverageCost        *string `json:"average_cost,omitempty"`
  Base               *string `json:"base,omitempty"`
  BaseCurrency       *string `json:"base_currency,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  NetProfit          *string `json:"net_profit,omitempty"`
  Profit             *string `json:"profit,omitempty"`
  Quote              *string `json:"quote,omitempty"`
  QuoteCurrency      *string `json:"quote_currency,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           *string `json:"strategy,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TradeId            int64   `json:"trade_id"`
  TradedAt           string  `json:"traded_at"`
  UserId             string  `json:"user_id"`
}

type PublicPaperPositionsUpdate struct {
  AverageCost        *string `json:"average_cost,omitempty"`
  Base               *string `json:"base,omitempty"`
  BaseCurrency       *string `json:"base_currency,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  NetProfit          *string `json:"net_profit,omitempty"`
  Profit             *string `json:"profit,omitempty"`
  Quote              *string `json:"quote,omitempty"`
  QuoteCurrency      *string `json:"quote_currency,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           *string `json:"strategy,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TradeId            *int64  `json:"trade_id"`
  TradedAt           *string `json:"traded_at,omitempty"`
  UserId             *string `json:"user_id,omitempty"`
}

type PublicPaperProfitsSelect struct {
  AverageCost        string  `json:"average_cost"`
  BaseCurrency       string  `json:"base_currency"`
  CreatedAt          string  `json:"created_at"`
  Exchange           string  `json:"exchange"`
  Fee                string  `json:"fee"`
  FeeCurrency        string  `json:"fee_currency"`
  FeeInUsd           *string `json:"fee_in_usd,omitempty"`
  Id                 string  `json:"id"`
  IsBuyer            bool    `json:"is_buyer"`
  IsFutures          bool    `json:"is_futures"`
  IsIsolated         bool    `json:"is_isolated"`
  IsMaker            bool    `json:"is_maker"`
  IsMargin           bool    `json:"is_margin"`
  NetProfit          string  `json:"net_profit"`
  NetProfitMargin    string  `json:"net_profit_margin"`
  Price              string  `json:"price"`
  Profit             string  `json:"profit"`
  ProfitMargin       string  `json:"profit_margin"`
  Quantity           string  `json:"quantity"`
  QuoteCurrency      string  `json:"quote_currency"`
  QuoteQuantity      string  `json:"quote_quantity"`
  Side               string  `json:"side"`
  Strategy           string  `json:"strategy"`
  StrategyInstanceId string  `json:"strategy_instance_id"`
  Symbol             string  `json:"symbol"`
  TradeId            int64   `json:"trade_id"`
  TradedAt           string  `json:"traded_at"`
  UserId             string  `json:"user_id"`
}

type PublicPaperProfitsInsert struct {
  AverageCost        *string `json:"average_cost,omitempty"`
  BaseCurrency       *string `json:"base_currency,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  Fee                *string `json:"fee,omitempty"`
  FeeCurrency        *string `json:"fee_currency,omitempty"`
  FeeInUsd           *string `json:"fee_in_usd,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsBuyer            *bool `json:"is_buyer,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMaker            *bool `json:"is_maker,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  NetProfit          *string `json:"net_profit,omitempty"`
  NetProfitMargin    *string `json:"net_profit_margin,omitempty"`
  Price              *string `json:"price,omitempty"`
  Profit             *string `json:"profit,omitempty"`
  ProfitMargin       *string `json:"profit_margin,omitempty"`
  Quantity           *string `json:"quantity,omitempty"`
  QuoteCurrency      *string `json:"quote_currency,omitempty"`
  QuoteQuantity      *string `json:"quote_quantity,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           *string `json:"strategy,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TradeId            int64   `json:"trade_id"`
  TradedAt           string  `json:"traded_at"`
  UserId             string  `json:"user_id"`
}

type PublicPaperProfitsUpdate struct {
  AverageCost        *string `json:"average_cost,omitempty"`
  BaseCurrency       *string `json:"base_currency,omitempty"`
  CreatedAt          *string `json:"created_at,omitempty"`
  Exchange           *string `json:"exchange,omitempty"`
  Fee                *string `json:"fee,omitempty"`
  FeeCurrency        *string `json:"fee_currency,omitempty"`
  FeeInUsd           *string `json:"fee_in_usd,omitempty"`
  Id                 *string `json:"id,omitempty,omitempty"`
  IsBuyer            *bool `json:"is_buyer,omitempty"`
  IsFutures          *bool `json:"is_futures,omitempty"`
  IsIsolated         *bool `json:"is_isolated,omitempty"`
  IsMaker            *bool `json:"is_maker,omitempty"`
  IsMargin           *bool `json:"is_margin,omitempty"`
  NetProfit          *string `json:"net_profit,omitempty"`
  NetProfitMargin    *string `json:"net_profit_margin,omitempty"`
  Price              *string `json:"price,omitempty"`
  Profit             *string `json:"profit,omitempty"`
  ProfitMargin       *string `json:"profit_margin,omitempty"`
  Quantity           *string `json:"quantity,omitempty"`
  QuoteCurrency      *string `json:"quote_currency,omitempty"`
  QuoteQuantity      *string `json:"quote_quantity,omitempty"`
  Side               *string `json:"side,omitempty"`
  Strategy           *string `json:"strategy,omitempty"`
  StrategyInstanceId *string `json:"strategy_instance_id,omitempty"`
  Symbol             *string `json:"symbol,omitempty"`
  TradeId            *int64  `json:"trade_id"`
  TradedAt           *string `json:"traded_at,omitempty"`
  UserId             *string `json:"user_id,omitempty"`
}
