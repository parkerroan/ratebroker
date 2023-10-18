/*
Package ratebroker manages rate limiting across various clients/servers, enabling control
over request frequency per user within a specific time frame and maximum request count.

One feature of ratebroker is its compatibility with message brokers to distribute rate
limits across multiple subscribers on the same topic/channel/stream. It uses an interface,
allowing adaptability with any message broker that follows the prescribed method signature.
This package includes support for Redis Streams, detailed at
https://redis.io/topics/streams-intro.

Without an external message broker, ratebroker employs a local limiter. Usage example:

	import (
		"time"
		"github.com/parkerroan/ratebroker"
	)

	func main() {
		// Set up ratebroker with a local limiter.
		rb := ratebroker.New(
			ratebroker.WithMaxRequests(10),
			ratebroker.WithWindow(10*time.Second),
		)

		allowed, detail := rb.TryAccept("userKey")
		// ... other code ...
	}

The package utilizes 'limiters' to enforce rate limits. By default, a ring buffer limiter is
used.

Two standalone limiters are also available, separate from the main ratebroker functionality,
for scenarios not requiring distributed or user-specific rate limits:

1. Ring: The default, a ring buffer limiter suitable where slight leniency is acceptable.
Details at https://github.com/parkerroan/ratebroker/limiter/ring.

2. Heap: A min-heap limiter for contexts needing higher precision, sorting requests by
timestamps. Though more accurate, it requires more resources. See
https://github.com/parkerroan/ratebroker/limiter/heap.

The choice between Ring and Heap depends on the precision and resource constraints of your
application.
*/
package ratebroker
