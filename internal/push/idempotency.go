package push

import (
	"crypto/sha256"
	"encoding/hex"
)

func DeriveIdempotencyKey(payload []byte) string {
	sum := sha256.Sum256(payload)
	return "content:" + hex.EncodeToString(sum[:])[:16]
}
