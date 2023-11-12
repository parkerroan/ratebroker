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
// This would be used if more accuracy is needed because the heap is sorted by the timestamp value
// not the order of the requests like with the ring buffer.
// This is also more expensive than the ring buffer.
type HeapLimiter struct {
	pq     priorityQueue
	window time.Duration
	size   int
	mutex  sync.Mutex
}

// NewHeapLimiterConstructorFunc returns a function that creates a new HeapLimiter.
func NewHeapLimiterConstructorFunc() func(int, time.Duration) Limiter {
	return func(size int, window time.Duration) Limiter {
		return NewHeapLimiter(size, window)
	}
}

// NewHeapLimiter returns a new HeapLimiter.
func NewHeapLimiter(size int, window time.Duration) *HeapLimiter {
	pq := make(priorityQueue, 0, size*10)
	heap.Init(&pq)
	return &HeapLimiter{
		pq:     pq,
		size:   size,
		window: window,
	}
}

// Accept implements the Limiter interface for the HeapLimiter.
// This is used when the request is accepted and added to the heap.
func (hl *HeapLimiter) Accept(now time.Time) {
	hl.mutex.Lock()
	defer hl.mutex.Unlock()
	hl.accept(now)
}

// TryAccept implements the Limiter interface for the HeapLimiter.
// This is used to check if the request is within the rate limits and if it is, it's added to the heap.
func (hl *HeapLimiter) TryAccept(now time.Time) bool {
	hl.mutex.Lock()
	defer hl.mutex.Unlock()

	if allowed := hl.try(now); allowed {
		hl.accept(now)
		return true
	}

	return false
}

// TryAccept checks if the request is within the rate limits and if it is, it's added to the heap.
// It returns RateLimitInfo with details about the current rate limit state.
func (hl *HeapLimiter) TryAcceptWithInfo(now time.Time) (bool, RateLimitInfo) {
	hl.mutex.Lock()
	defer hl.mutex.Unlock()

	// First, remove any outdated requests outside of the current window.
	for hl.pq.Len() > 0 && now.Sub(hl.pq[0].timestamp) > hl.window {
		heap.Pop(&hl.pq)
	}

	// Prepare the information to return.
	info := RateLimitInfo{
		Remaining: hl.size - hl.pq.Len(), // Remaining requests that can be accepted.
		Limit:     hl.size,               // Maximum number of requests in the window.
	}

	// If the heap is not full, we can accept the request.
	if info.Remaining > 0 {
		// Accept the request by pushing it to the priority queue.
		hl.accept(now)

		// Update the remaining number of requests after accepting the current one.
		info.Remaining--

		// If there are multiple requests, calculate the time until the rate limit resets
		// based on the oldest timestamp in the heap.
		if hl.pq.Len() > 1 {
			oldest := hl.pq[0].timestamp
			info.Reset = oldest.Add(hl.window).Sub(now)
		} else {
			// If it's the only request, the reset time is the window duration.
			info.Reset = hl.window
		}

		return true, info
	}

	// If we're here, it means the limit has been reached. Calculate the reset time based on the oldest request.
	oldest := hl.pq[0].timestamp
	info.Reset = oldest.Add(hl.window).Sub(now)

	return false, info
}

// LimitDetails returns the size and window of the limiter.
func (hl *HeapLimiter) LimitDetails() (int, time.Duration) {
	return hl.size, hl.window
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
	if hl.pq.Len() >= hl.size {
		// The heap is full, i.e., we've reached the rate limit.
		return false
	}

	return true
}
