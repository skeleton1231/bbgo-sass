package main

import (
	"encoding/json"
	"fmt"
	"time"

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

func (sc *SupabaseClient) LoadUserContainers() ([]*UserContainer, error) {
	data, _, err := sc.client.From("user_containers").Select("*", "", false).Execute()
	if err != nil {
		return nil, fmt.Errorf("load user containers: %w", err)
	}

	var rows []PublicUserContainersSelect
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}

	users := make([]*UserContainer, len(rows))
	for i, r := range rows {
		uc := &UserContainer{UserID: r.UserId, Mode: r.Mode, Status: r.Status}
		if uc.Mode == "" {
			uc.Mode = ModeLive
		}
		if r.Strategies != nil {
			if b, err := json.Marshal(r.Strategies); err == nil {
				json.Unmarshal(b, &uc.Strategies)
			}
		}
		users[i] = uc
	}
	return users, nil
}

func (sc *SupabaseClient) UpsertUser(uc *UserContainer) error {
	now := time.Now().UTC().Format(time.RFC3339)
	row := PublicUserContainersInsert{
		UserId:     uc.UserID,
		Mode:       ptrStr(uc.Mode),
		Status:     ptrStr(uc.Status),
		Strategies: uc.Strategies,
		CreatedAt:  ptrStr(now),
		UpdatedAt:  ptrStr(now),
	}
	_, _, err := sc.client.From("user_containers").Upsert(row, "user_id,mode", "", "").Execute()
	if err != nil {
		return fmt.Errorf("upsert user %s: %w", uc.UserID, err)
	}
	return nil
}

func (sc *SupabaseClient) UpsertCredential(cred ExchangeCredential) error {
	row := PublicExchangeCredentialsInsert{
		Id:                  &cred.ID,
		UserId:              cred.UserID,
		Exchange:            cred.Exchange,
		ApiKeyEncrypted:     cred.APIKeyEncrypted,
		ApiSecretEncrypted:  cred.APISecretEncrypted,
		PassphraseEncrypted: &cred.PassphraseEncrypted,
		IsTestnet:           &cred.IsTestnet,
		IsVerified:          &cred.IsVerified,
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
			Select("symbol,side,price,quantity,fee,traded_at", "", false).
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
