package limiter_test

import (
	"testing"
	"time"

	"github.com/parkerroan/ratebroker/limiter"
	"github.com/stretchr/testify/assert"
)

func BenchmarkUnitRingLimiter(b *testing.B) {
	rl := limiter.NewRingLimiter(10, time.Second)
	now := time.Now()

	for i := 0; i < b.N; i++ {
		rl.TryAccept(now)
	}
}

func BenchmarkUnitRingLimiter_TryAcceptV2(b *testing.B) {
	rl := limiter.NewRingLimiter(10000, time.Second)
	now := time.Now()

	for i := 0; i < b.N; i++ {
		rl.TryAcceptWithInfo(now)
	}
}

func TestUnitRingLimiter(t *testing.T) {
	// Create a new RingLimiter with size 3 and window 1 second.
	rl := limiter.NewRingLimiter(3, time.Second)

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

func TestUnitRingLimiter_TryAcceptV2(t *testing.T) {
	// Create a new RingLimiter with size 3 and window 1 second.
	rl := limiter.NewRingLimiter(3, time.Second)

	// Check that the first 3 requests are allowed.
	ok, info := rl.TryAcceptWithInfo(time.Now())
	if !ok {
		t.Error("First request should be allowed")
	}
	assert.Equal(t, 2, info.Remaining)

	ok, info = rl.TryAcceptWithInfo(time.Now())
	if !ok {
		t.Error("Second request should be allowed")
	}
	assert.Equal(t, 1, info.Remaining)

	ok, info = rl.TryAcceptWithInfo(time.Now())
	if !ok {
		t.Error("Third request should be allowed")
	}
	assert.Equal(t, 0, info.Remaining)

	// Check that the fourth request is not allowed.
	ok, _ = rl.TryAcceptWithInfo(time.Now())
	if ok {
		t.Error("Fourth request should not be allowed")
	}

	// Wait for 1 second and check that the fourth request is now allowed.
	time.Sleep(time.Second)
	ok, _ = rl.TryAcceptWithInfo(time.Now())
	if !ok {
		t.Error("Fourth request should be allowed after waiting 1 second")
	}
}
