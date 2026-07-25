package id

import (
	"crypto/rand"
	"encoding/hex"
)

// New returns "<prefix>-<12 hex chars>", e.g. "plan-0a1b2c3d4e5f".
func New(prefix string) string {
	var b [6]byte
	// rand.Read never returns an error
	rand.Read(b[:])
	return prefix + "-" + hex.EncodeToString(b[:])
}
