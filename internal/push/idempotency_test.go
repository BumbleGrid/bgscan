package push

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestDeriveIdempotencyKey_nilPayload(t *testing.T) {
	key := DeriveIdempotencyKey(nil)
	sum := sha256.Sum256(nil)
	want := "content:" + hex.EncodeToString(sum[:])[:16]
	if key != want {
		t.Fatalf("DeriveIdempotencyKey(nil) = %q, want %q", key, want)
	}
}

func TestDeriveIdempotencyKey_hello(t *testing.T) {
	payload := []byte("hello")
	key := DeriveIdempotencyKey(payload)
	sum := sha256.Sum256(payload)
	want := "content:" + hex.EncodeToString(sum[:])[:16]
	if key != want {
		t.Fatalf("DeriveIdempotencyKey(hello) = %q, want %q", key, want)
	}
}

func TestDeriveIdempotencyKey_distinctPayloads(t *testing.T) {
	keyA := DeriveIdempotencyKey([]byte("alpha"))
	keyB := DeriveIdempotencyKey([]byte("beta"))
	if keyA == keyB {
		t.Fatal("expected distinct keys for distinct payloads")
	}
}
