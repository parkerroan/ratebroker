//go:build unit

package limiter

import (
	"testing"
	"time"
)

func TestRingLimiter(t *testing.T) {
	// Create a new RingLimiter with size 3 and window 1 second.
	rl := NewRingLimiter(3, time.Second)

	// Check that the first 3 requests are allowed.
	if !rl.TryAccept(time.Now()) {
		t.Error("First request should be allowed")
	}
	if !rl.TryAccept(time.Now()) {
		t.Error("Second request should be allowed")
	}
	if !rl.TryAccept(time.Now()) {
		t.Error("Third request should be allowed")
	}

	// Check that the fourth request is not allowed.
	if rl.TryAccept(time.Now()) {
		t.Error("Fourth request should not be allowed")
	}

	// Wait for 1 second and check that the fourth request is now allowed.
	time.Sleep(time.Second)
	if !rl.TryAccept(time.Now()) {
		t.Error("Fourth request should be allowed after waiting 1 second")
	}
}
