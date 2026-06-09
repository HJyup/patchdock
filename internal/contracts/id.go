package contracts

import (
	"crypto/rand"
	"encoding/hex"
)

// newID returns "<prefix>-<12 hex chars>", e.g. "plan-0a1b2c3d4e5f".
func newID(prefix string) string {
	var b [6]byte
	rand.Read(b[:]) // !!! "Never return an error on all but legacy Linux systems"
	return prefix + "-" + hex.EncodeToString(b[:])
}
