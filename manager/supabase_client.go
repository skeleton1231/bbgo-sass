package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/supabase-community/postgrest-go"
	supabase "github.com/supabase-community/supabase-go"
)

type SupabaseClient struct {
	client *supabase.Client
}

func NewSupabaseClient(supabaseURL, supabaseKey string) (*SupabaseClient, error) {
	client, err := supabase.NewClient(supabaseURL, supabaseKey, &supabase.ClientOptions{})
	if err != nil {
		return nil, fmt.Errorf("create supabase client: %w", err)
	}
	return &SupabaseClient{client: client}, nil
}

func (sc *SupabaseClient) UpsertCredential(cred ExchangeCredential) error {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	row := PublicExchangeCredentialsInsert{
		Id:                  &id,
		UserId:              cred.UserID,
		Exchange:            cred.Exchange,
		ApiKeyEncrypted:     cred.APIKeyEncrypted,
		ApiSecretEncrypted:  cred.APISecretEncrypted,
		PassphraseEncrypted: &cred.PassphraseEncrypted,
		IsTestnet:           &cred.IsTestnet,
		IsVerified:          &cred.IsVerified,
		CreatedAt:           &now,
	}
	_, _, err := sc.client.From("exchange_credentials").Upsert(row, "user_id,exchange,is_testnet", "", "").Execute()
	if err != nil {
		return fmt.Errorf("sync credential for user %s %s: %w", cred.UserID, cred.Exchange, err)
	}
	return nil
}

func (sc *SupabaseClient) DeleteCredential(userID, exchange string, isTestnet bool) error {
	_, _, err := sc.client.From("exchange_credentials").
		Delete("", "").
		Eq("user_id", userID).
		Eq("exchange", exchange).
		Eq("is_testnet", fmt.Sprintf("%t", isTestnet)).
		Execute()
	if err != nil {
		return fmt.Errorf("delete credential %s for user %s: %w", exchange, userID, err)
	}
	return nil
}

const pnlPageSize = 1000

func (sc *SupabaseClient) GetTradesForPnL(userID string) ([]BBGoTrade, error) {
	var allTrades []BBGoTrade
	offset := 0

	for {
		from := offset
		to := offset + pnlPageSize - 1
		data, _, err := sc.client.From("trades").
			Select("symbol,side,price,quantity,fee,traded_at,exchange,is_buyer,is_maker,is_futures,is_margin,order_id,trade_id,strategy", "", false).
			Eq("user_id", userID).
			Order("traded_at", &postgrest.OrderOpts{Ascending: true}).
			Range(from, to, "").
			Execute()
		if err != nil {
			return nil, fmt.Errorf("fetch trades for pnl: %w", err)
		}

		var rows []PublicTradesSelect
		if err := json.Unmarshal(data, &rows); err != nil {
			return nil, fmt.Errorf("decode trades for pnl: %w", err)
		}

		for _, r := range rows {
			tradedAt := ""
			if r.TradedAt != nil {
				tradedAt = *r.TradedAt
			}
			allTrades = append(allTrades, BBGoTrade{
				Symbol: r.Symbol, Side: r.Side, Price: flexString(r.Price),
				Quantity: flexString(r.Quantity), Fee: flexString(r.Fee), TradedAt: tradedAt,
				Exchange: r.Exchange, IsBuyer: r.IsBuyer, IsMaker: r.IsMaker,
				StrategyID: r.Strategy,
				ID: parseUintOrZero(r.TradeId), OrderID: parseUintOrZero(r.OrderId),
			})
		}

		if len(rows) < pnlPageSize {
			break
		}
		offset += len(rows)
	}
	return allTrades, nil
}

func ptrStr(s string) *string { return &s }

func parseUintOrZero(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
