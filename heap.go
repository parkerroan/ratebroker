package ratebroker

import (
	"container/heap"
	"time"
)

// Item represents a single object in the heap.
type Item struct {
	timestamp time.Time // The timestamp of the request
	index     int       // The index is needed by update and is maintained by the heap.Interface methods.
}

// PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want a min heap, so we use Less here. The earliest timestamp will be the root.
	return pq[i].timestamp.Before(pq[j].timestamp)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
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
	pq     PriorityQueue
	window time.Duration
}

// NewHeapLimiter creates a HeapLimiter.
func NewHeapLimiter(size int, window time.Duration) *HeapLimiter {
	pq := make(PriorityQueue, 0, size)
	heap.Init(&pq)
	return &HeapLimiter{
		pq:     pq,
		window: window,
	}
}

// TryAccept implements the Limiter interface for the HeapLimiter.
func (hl *HeapLimiter) TryAccept(now time.Time) bool {
	// Remove the timestamps that are out of the window range.
	for hl.pq.Len() > 0 && now.Sub(hl.pq[0].timestamp) > hl.window {
		heap.Pop(&hl.pq)
	}

	// Check if there's room for more requests.
	if hl.pq.Len() >= cap(hl.pq) {
		// The heap is full, i.e., we've reached the rate limit.
		return false
	}

	// There's room for another request, so we add the new timestamp.
	item := &Item{
		timestamp: now,
	}
	heap.Push(&hl.pq, item)

	return true
}
