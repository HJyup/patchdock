package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID returns "<prefix>-<12 hex chars>", e.g. "plan-0a1b2c3d4e5f".
func NewID(prefix string) string {
	var b [6]byte
	// The default Reader uses operating system APIs that are
	// documented to never return an error on all but legacy Linux systems.
	rand.Read(b[:])
	return prefix + "-" + hex.EncodeToString(b[:])
}
