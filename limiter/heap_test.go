package limiter_test

import (
	"testing"
	"time"

	"github.com/parkerroan/ratebroker/limiter"
)

func BenchmarkUnitHeapLimiter(b *testing.B) {
	hl := limiter.NewHeapLimiter(10, time.Second)
	now := time.Now()

	for i := 0; i < b.N; i++ {
		hl.TryAccept(now)
	}
}

func BenchmarkUnitHeapLimiter_TryAcceptV2(b *testing.B) {
	hl := limiter.NewHeapLimiter(10, time.Second)
	now := time.Now()

	for i := 0; i < b.N; i++ {
		hl.TryAcceptWithInfo(now)
	}
}

func TestUnitHeapLimiter_TryAccept(t *testing.T) {
	// Create a new RingLimiter with size 3 and window 1 second.
	hl := limiter.NewHeapLimiter(3, time.Second)

	// Check that the first 3 requests are allowed.
	if !hl.TryAccept(time.Now()) {
		t.Error("First request should be allowed")
	}
	if !hl.TryAccept(time.Now()) {
		t.Error("Second request should be allowed")
	}
	if !hl.TryAccept(time.Now()) {
		t.Error("Third request should be allowed")
	}

	// Check that the fourth request is not allowed.
	if hl.TryAccept(time.Now()) {
		t.Error("Fourth request should not be allowed")
	}

	// Wait for 1 second and check that the fourth request is now allowed.
	time.Sleep(time.Second)
	if !hl.TryAccept(time.Now()) {
		t.Error("Fourth request should be allowed after waiting 1 second")
	}
}
