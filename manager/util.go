package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func generateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}
