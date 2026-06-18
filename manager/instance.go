package main

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/c9s/bbgo/pkg/instanceid"
)

// StrategyInstance represents one strategy running in its own container.
type StrategyInstance struct {
	InstanceID    string              `json:"instanceId"`
	UserID        string              `json:"userId"`
	Mode          string              `json:"mode"`
	Strategy      string              `json:"strategy"`
	Exchange      string              `json:"exchange"`
	Symbol        string              `json:"symbol"`
	Config        json.RawMessage     `json:"config"`
	Name          string              `json:"name"`
	CrossExchange bool                `json:"crossExchange"`
	Sessions      []SessionRoleConfig `json:"sessions,omitempty"`
	FuturesConfig *FuturesConfig      `json:"futuresConfig,omitempty"`
	RiskConfig    *RiskConfig         `json:"riskConfig,omitempty"`
	LastError     string              `json:"lastError,omitempty"`
	LastErrorAt   string              `json:"lastErrorAt,omitempty"`
}

// computeInstanceID delegates to the shared instanceid package for bbgo-canonical IDs.
func computeInstanceID(strategy, symbol string, config json.RawMessage) string {
	return instanceid.Compute(strategy, symbol, config)
}

// instanceSlug returns a Docker-valid container name for an instance.
// Format: "bbgo-{userID:8}-{mode}-{sanitizedInstanceID}"
// Uses first 8 chars of userID to keep names short enough for Docker DNS.
func instanceSlug(userID, mode, instanceID string) string {
	shortUser := userID
	if len(shortUser) > 8 {
		shortUser = shortUser[:8]
	}
	slug := "bbgo-" + shortUser + "-" + mode + "-" + sanitizeForDocker(instanceID)
	if len(slug) > 128 {
		slug = slug[:128]
	}
	return slug
}

var dockerInvalid = regexp.MustCompile(`[^a-z0-9-]`)

func sanitizeForDocker(s string) string {
	s = strings.ToLower(s)
	s = dockerInvalid.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}

// instanceDirName returns the directory name for an instance within the user's data dir.
func instanceDirName(instanceID string) string {
	return sanitizeForDocker(instanceID)
}
