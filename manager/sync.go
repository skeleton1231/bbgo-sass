package main

import (
	"log"
)

type Syncer struct {
	users    *UserContainerManager
	supa     *SupabaseClient
	creds    *CredentialStore
	notifier *Notifier
}

func NewSyncer(users *UserContainerManager, supa *SupabaseClient) *Syncer {
	return &Syncer{users: users, supa: supa}
}

func NewSyncerWithCreds(users *UserContainerManager, supa *SupabaseClient, creds *CredentialStore) *Syncer {
	s := NewSyncer(users, supa)
	s.creds = creds
	return s
}

func (s *Syncer) SetNotifier(n *Notifier) {
	s.notifier = n
}

func (s *Syncer) LoadUsersFromSupabase() ([]*UserContainer, error) {
	return s.supa.LoadUserContainers()
}

func (s *Syncer) UpsertUser(uc *UserContainer) {
	if err := s.supa.UpsertUser(uc); err != nil {
		log.Printf("upsert user %s: %v", uc.UserID, err)
	}
}

func (s *Syncer) SyncUser(userID, mode string) {
	uc, ok := s.users.Get(userID, mode)
	if !ok {
		return
	}
	s.UpsertUser(uc)
}

func (s *Syncer) SyncAll() {
	for _, uc := range s.users.ListUsers() {
		s.UpsertUser(uc)
	}
}

func (s *Syncer) SyncCredential(cred ExchangeCredential) {
	if s.creds == nil {
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

func (s *Syncer) markCredentialsVerified(uc *UserContainer) {
	if s.creds == nil {
		return
	}
	creds, err := s.creds.List(uc.UserID)
	if err != nil {
		return
	}
	for _, c := range creds {
		if c.IsVerified {
			continue
		}
		exchanges := []string{}
		for _, strat := range uc.Strategies {
			if strat.CrossExchange {
				for _, sr := range strat.Sessions {
					exchanges = append(exchanges, sr.Exchange)
				}
			} else if strat.Exchange != "" {
				exchanges = append(exchanges, strat.Exchange)
			}
		}
		for _, ex := range exchanges {
			if ex == c.Exchange {
				c.IsVerified = true
				s.creds.Update(uc.UserID, c)
				log.Printf("credential %s for user %s marked as verified", c.Exchange, uc.UserID)
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
