package heap

import (
	"container/heap"
	"sync"
	"time"
)

// HeapLimiter is an implementation of the Limiter interface using a min-heap.
type HeapLimiter struct {
	pq     priorityQueue
	window time.Duration
	mutex  sync.Mutex
}


// NewHeapLimiter creates a HeapLimiter.
func NewHeapLimiter(size int, window time.Duration) *HeapLimiter {
	pq := make(priorityQueue, 0, size)
	heap.Init(&pq)
	return &HeapLimiter{
		pq:     pq,
		window: window,
	}
}

// TryAccept implements the Limiter interface for the HeapLimiter.
func (hl *HeapLimiter) Try(now time.Time) bool {
	hl.mutex.Lock()
	defer hl.mutex.Unlock()
	// Remove the timestamps that are out of the window range.
	for hl.pq.Len() > 0 && now.Sub(hl.pq[0].timestamp) > hl.window {
		heap.Pop(&hl.pq)
	}

	// Check if there's room for more requests.
	if hl.pq.Len() >= cap(hl.pq) {
		// The heap is full, i.e., we've reached the rate limit.
		return false
	}

	// // There's room for another request, so we add the new timestamp.
	// item := &item{
	// 	timestamp: now,
	// }
	// heap.Push(&hl.pq, item)

	return true
}

func (hl *HeapLimiter) Accept(now time.Time) {
	hl.mutex.Lock()
	defer hl.mutex.Unlock()
	item := &item{
		timestamp: now,
	}
	heap.Push(&hl.pq, item)
}
