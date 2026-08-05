package users

import (
	"context"
	"testing"
	"time"
)

// An already-expired expiresAt must be rejected before it reaches the client:
// Aerospike reads a non-positive TTL as "use the namespace default", which
// would quietly grant the token the default lifetime instead of dropping it.
func TestSaveRefreshToken_RejectsExpired(t *testing.T) {
	// Nil client is safe here -- the guard returns before any call on it.
	store := NewAerospikeStore(nil, "test", "refresh_tokens")

	for name, expiresAt := range map[string]time.Time{
		"past": time.Now().Add(-time.Hour),
		"now":  time.Now(),
	} {
		if err := store.SaveRefreshToken(context.Background(), "jti", "user", expiresAt); err == nil {
			t.Errorf("%s: expected error for expired token, got nil", name)
		}
	}
}
