package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type ExchangeCredential struct {
	ID                  string `json:"id"`
	UserID              string `json:"user_id"`
	Exchange            string `json:"exchange"`
	APIKeyEncrypted     string `json:"api_key_encrypted"`
	APISecretEncrypted  string `json:"api_secret_encrypted"`
	PassphraseEncrypted string `json:"passphrase_encrypted,omitempty"`
	IsTestnet           bool   `json:"is_testnet"`
	IsVerified          bool   `json:"is_verified"`
}

type CredentialStore struct {
	mu     sync.RWMutex
	dir    string
	crypto *Encryptor
}

func NewCredentialStore(dataDir string, enc *Encryptor) *CredentialStore {
	return &CredentialStore{dir: dataDir, crypto: enc}
}

func (cs *CredentialStore) filePath(userID string) string {
	return filepath.Join(cs.dir, userID, "credentials.json")
}

func (cs *CredentialStore) loadAll(userID string) ([]ExchangeCredential, error) {
	data, err := os.ReadFile(cs.filePath(userID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var creds []ExchangeCredential
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return creds, nil
}

func (cs *CredentialStore) saveAll(userID string, creds []ExchangeCredential) error {
	dir := filepath.Dir(cs.filePath(userID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cs.filePath(userID), data, 0o600)
}

// Upsert inserts a credential or replaces an existing one for the same exchange.
func (cs *CredentialStore) Upsert(cred ExchangeCredential) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	creds, err := cs.loadAll(cred.UserID)
	if err != nil {
		return err
	}
	for i, c := range creds {
		if c.Exchange == cred.Exchange {
			creds[i] = cred
			return cs.saveAll(cred.UserID, creds)
		}
	}
	creds = append(creds, cred)
	return cs.saveAll(cred.UserID, creds)
}

func (cs *CredentialStore) List(userID string) ([]ExchangeCredential, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.loadAll(userID)
}

func (cs *CredentialStore) Update(userID string, cred ExchangeCredential) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	creds, err := cs.loadAll(userID)
	if err != nil {
		return err
	}
	for i, c := range creds {
		if c.ID == cred.ID {
			creds[i] = cred
			return cs.saveAll(userID, creds)
		}
	}
	return fmt.Errorf("credential %s not found", cred.ID)
}

func (cs *CredentialStore) Delete(userID, id string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	creds, err := cs.loadAll(userID)
	if err != nil {
		return err
	}
	filtered := make([]ExchangeCredential, 0, len(creds))
	for _, c := range creds {
		if c.ID != id {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == len(creds) {
		return fmt.Errorf("credential %s not found", id)
	}
	return cs.saveAll(userID, filtered)
}

func (cs *CredentialStore) GetDecrypted(userID, exchange string) (apiKey, apiSecret, passphrase string, err error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	creds, err := cs.loadAll(userID)
	if err != nil {
		return "", "", "", err
	}
	for _, c := range creds {
		if c.Exchange == exchange {
			apiKey, err = cs.crypto.Decrypt(c.APIKeyEncrypted)
			if err != nil {
				return "", "", "", fmt.Errorf("decrypt api key: %w", err)
			}
			apiSecret, err = cs.crypto.Decrypt(c.APISecretEncrypted)
			if err != nil {
				return "", "", "", fmt.Errorf("decrypt api secret: %w", err)
			}
			if c.PassphraseEncrypted != "" {
				passphrase, err = cs.crypto.Decrypt(c.PassphraseEncrypted)
				if err != nil {
					return "", "", "", fmt.Errorf("decrypt passphrase: %w", err)
				}
			}
			return apiKey, apiSecret, passphrase, nil
		}
	}
	return "", "", "", fmt.Errorf("no credentials for exchange %s", exchange)
}
