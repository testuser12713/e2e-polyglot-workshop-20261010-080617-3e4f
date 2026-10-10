package auth

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPasswordVerifies(t *testing.T) {
	const password = "s3cret-passphrase"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == password {
		t.Fatal("HashPassword() returned the plaintext password")
	}
	if !VerifyPassword(hash, password) {
		t.Error("VerifyPassword() rejected the correct password")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Error("VerifyPassword() accepted a wrong password")
	}
}

func TestHashPasswordUsesExplicitCost(t *testing.T) {
	hash, err := HashPassword("cost-check")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("bcrypt.Cost() error = %v", err)
	}
	if cost != BcryptCost {
		t.Errorf("hash cost = %d, want explicit BcryptCost %d", cost, BcryptCost)
	}
}

func TestHashPasswordIsSalted(t *testing.T) {
	first, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	second, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if first == second {
		t.Error("hashing the same password twice produced identical hashes (no salt)")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	if VerifyPassword("not-a-bcrypt-hash", "anything") {
		t.Error("VerifyPassword() accepted a malformed hash")
	}
}

// TestBootstrapSkipsWithoutCredentials proves the startup hook is a no-op when
// the initial employee variables are absent: it returns before touching the
// database, so a nil pool is safe here.
func TestBootstrapSkipsWithoutCredentials(t *testing.T) {
	if err := BootstrapInitialEmployee(context.Background(), nil, "", ""); err != nil {
		t.Errorf("BootstrapInitialEmployee(empty, empty) = %v, want nil", err)
	}
	if err := BootstrapInitialEmployee(context.Background(), nil, "   ", "secret"); err != nil {
		t.Errorf("BootstrapInitialEmployee(blank email) = %v, want nil", err)
	}
}

func TestRateLimiterBlocksAfterMaxFailures(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)
	const key = "203.0.113.7"

	for attempt := 1; attempt <= 3; attempt++ {
		if !limiter.Allow(key) {
			t.Fatalf("attempt %d blocked before the failure allowance was reached", attempt)
		}
		limiter.RecordFailure(key)
	}
	if limiter.Allow(key) {
		t.Error("limiter allowed an attempt after the failure allowance was exhausted")
	}
}

func TestRateLimiterIsPerClient(t *testing.T) {
	limiter := NewRateLimiter(1, time.Minute)
	limiter.RecordFailure("client-a")

	if limiter.Allow("client-a") {
		t.Error("client-a should be blocked")
	}
	if !limiter.Allow("client-b") {
		t.Error("client-b must not be affected by client-a's failures")
	}
}

func TestRateLimiterResetClearsFailures(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	limiter.RecordFailure("client")
	limiter.RecordFailure("client")
	limiter.Reset("client")

	if !limiter.Allow("client") {
		t.Error("Reset() did not clear the recorded failures")
	}
}

func TestRateLimiterWindowExpires(t *testing.T) {
	current := time.Now()
	limiter := NewRateLimiter(1, time.Minute)
	limiter.now = func() time.Time { return current }
	limiter.RecordFailure("client")
	if limiter.Allow("client") {
		t.Fatal("client should be blocked inside the window")
	}

	current = current.Add(2 * time.Minute)
	if !limiter.Allow("client") {
		t.Error("client should be allowed again once the window has passed")
	}
}
