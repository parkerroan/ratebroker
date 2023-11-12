package limiter_test

import (
	"testing"
	"time"

	"github.com/parkerroan/ratebroker/limiter"
	"github.com/stretchr/testify/assert"
)

func BenchmarkUnitTokenLimiter(b *testing.B) {
	tl := limiter.NewTokenLimiter(10, time.Second)
	now := time.Now()

	for i := 0; i < b.N; i++ {
		tl.TryAccept(now)
	}
}

func BenchmarkUnitTokenLimiter_TryAcceptV2(b *testing.B) {
	tl := limiter.NewTokenLimiter(10, time.Second)
	now := time.Now()

	for i := 0; i < b.N; i++ {
		tl.TryAcceptWithInfo(now)
	}
}

func TestUnitTokenLimiter_TryAcceptV2(t *testing.T) {
	// Create a new RingLimiter with size 3 and window 1 second.
	tl := limiter.NewTokenLimiter(3, time.Second)

	// Check that the first 3 requests are allowed.
	ok, info := tl.TryAcceptWithInfo(time.Now())
	if !ok {
		t.Error("First request should be allowed")
	}
	assert.Equal(t, 2, info.Remaining)

	ok, info = tl.TryAcceptWithInfo(time.Now())
	if !ok {
		t.Error("Second request should be allowed")
	}
	assert.Equal(t, 1, info.Remaining)

	ok, info = tl.TryAcceptWithInfo(time.Now())
	if !ok {
		t.Error("Third request should be allowed")
	}
	assert.Equal(t, 0, info.Remaining)

	// Check that the fourth request is not allowed.
	ok, _ = tl.TryAcceptWithInfo(time.Now())
	if ok {
		t.Error("Fourth request should not be allowed")
	}

	// Wait for 1 second and check that the fourth request is now allowed.
	time.Sleep(time.Second)
	ok, _ = tl.TryAcceptWithInfo(time.Now())
	if !ok {
		t.Error("Fourth request should be allowed after waiting 1 second")
	}
}

func TestRateLimiter_LimitDetails(t *testing.T) {
	size := 2
	window := 500 * time.Millisecond // a half of a second

	limiter := limiter.NewTokenLimiter(size, window)

	actualSize, actualWindow := limiter.LimitDetails()
	if actualSize != size || actualWindow != window {
		t.Errorf("Expected size: %d and window: %v, but got size: %d and window: %v", size, window, actualSize, actualWindow)
	}
}

// Additional tests should include more scenarios, especially edge cases.
// You might also want to write benchmarks to test the performance of your limiter under load.
