package limiter

import (
	"container/heap"
	"sync"
	"time"
)

// item represents a single object in the heap.
type item struct {
	timestamp time.Time // The timestamp of the request
	index     int       // The index is needed by update and is maintained by the heap.Interface methods.
}

// priorityQueue implements heap.Interface and holds Items.
type priorityQueue []*item

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// We want a min heap, so we use Less here. The earliest timestamp will be the root.
	return pq[i].timestamp.Before(pq[j].timestamp)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// HeapLimiter is an implementation of the Limiter interface using a min-heap.
type HeapLimiter struct {
	pq     priorityQueue
	window time.Duration
	mutex  sync.Mutex
}

func NewHeapLimiterConstructorFunc() func(int, time.Duration) Limiter {
	return func(size int, window time.Duration) Limiter {
		return NewHeapLimiter(size, window)
	}
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
	return hl.try(now)
}

func (hl *HeapLimiter) Accept(now time.Time) {
	hl.mutex.Lock()
	defer hl.mutex.Unlock()
	hl.accept(now)
}

// TryAccept implements the Limiter interface for the HeapLimiter.
func (hl *HeapLimiter) TryAccept(now time.Time) bool {
	hl.mutex.Lock()
	defer hl.mutex.Unlock()

	if allowed := hl.try(now); allowed {
		hl.accept(now)
		return true
	}

	return false
}

func (hl *HeapLimiter) accept(now time.Time) {
	item := &item{
		timestamp: now,
	}
	heap.Push(&hl.pq, item)
}

// TryAccept implements the Limiter interface for the HeapLimiter.
func (hl *HeapLimiter) try(now time.Time) bool {
	// Remove the timestamps that are out of the window range.
	for hl.pq.Len() > 0 && now.Sub(hl.pq[0].timestamp) > hl.window {
		heap.Pop(&hl.pq)
	}

	// Check if there's room for more requests.
	if hl.pq.Len() >= cap(hl.pq) {
		// The heap is full, i.e., we've reached the rate limit.
		return false
	}

	return true
}
