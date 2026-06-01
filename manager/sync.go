package main

import (
	"log"
)

type Syncer struct {
	supa     *SupabaseClient
	creds    *CredentialStore
	notifier *Notifier
}

func NewSyncer(supa *SupabaseClient) *Syncer {
	return &Syncer{supa: supa}
}

func NewSyncerWithCreds(supa *SupabaseClient, creds *CredentialStore) *Syncer {
	s := NewSyncer(supa)
	s.creds = creds
	return s
}

func (s *Syncer) SetNotifier(n *Notifier) {
	s.notifier = n
}

func (s *Syncer) SyncCredential(cred ExchangeCredential) {
	if s.supa == nil {
		return
	}
	if err := s.supa.UpsertCredential(cred); err != nil {
		log.Printf("sync credential for user %s %s: %v", cred.UserID, cred.Exchange, err)
	} else {
		log.Printf("synced credential %s for user %s", cred.Exchange, cred.UserID)
	}
}

func (s *Syncer) DeleteCredential(userID, exchange string, isTestnet bool) {
	if s.creds == nil {
		return
	}
	if err := s.supa.DeleteCredential(userID, exchange, isTestnet); err != nil {
		log.Printf("delete credential for user %s %s: %v", userID, exchange, err)
	} else {
		log.Printf("deleted credential %s (testnet=%v) for user %s from supabase", exchange, isTestnet, userID)
	}
}

func (s *Syncer) MarkCredentialsVerified(userID, mode string, strategies []StrategyEntry) {
	if s.creds == nil {
		return
	}
	creds, err := s.creds.List(userID)
	if err != nil {
		return
	}
	wantTestnet := mode == ModePaper

	exchanges := []string{}
	for _, strat := range strategies {
		if strat.CrossExchange {
			for _, sr := range strat.Sessions {
				exchanges = append(exchanges, sr.Exchange)
			}
		} else if strat.Exchange != "" {
			exchanges = append(exchanges, strat.Exchange)
		}
	}

	for _, c := range creds {
		if c.IsVerified {
			continue
		}
		if c.IsTestnet != wantTestnet {
			continue
		}
		for _, ex := range exchanges {
			if ex == c.Exchange {
				c.IsVerified = true
				s.creds.Update(userID, c)
				log.Printf("credential %s (%s) for user %s marked as verified", c.Exchange, modeLabel(c.IsTestnet), userID)
				s.SyncCredential(c)
				break
			}
		}
	}
}

func (s *Syncer) GetTradesForPnL(userID string) ([]BBGoTrade, error) {
	return s.supa.GetTradesForPnL(userID)
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
