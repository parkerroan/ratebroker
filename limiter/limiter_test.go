package limiter_test

import (
	"testing"
	"time"

	"github.com/parkerroan/ratebroker/limiter"
)

func BenchmarkRingLimiter(b *testing.B) {
	rl := limiter.NewRingLimiter(10, time.Second)
	now := time.Now()

	for i := 0; i < b.N; i++ {
		rl.Try(now)
		rl.Accept(now)
	}
}

func BenchmarkHeapLimiter(b *testing.B) {
	hl := limiter.NewHeapLimiter(10, time.Second)
	now := time.Now()

	for i := 0; i < b.N; i++ {
		hl.Try(now)
		hl.Accept(now)
	}
}
