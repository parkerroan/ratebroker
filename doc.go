/*
Package ratebroker provides a rate limiting broker that can be used across many clients/servers
to limit the number of requests per user over a given time window.

The ratebroker has the option to use a message broker to distribute the rate limiting to all
clients/servers subscribed to the same topic/channel/stream.

# If no message broker is provided, the ratebroker will use a local limiter
Example:

	import (
		"time"
		"github.com/parkerroan/ratebroker"
	)

	// Create a new ratebroker with a local limiter
	rb := ratebroker.New(
		ratebroker.WithMaxRequests(10),
		ratebroker.WithWindow(10*time.Second),
	)

The ratebroker uses a limiter to enforce rate limits. The default limiter is a ring buffer

The repo provides 2 limiters and each can be used without the ratebroker
if you don't need to distribute the limit or implement per user limiting:
- Ring  (https://github.com/parkerroan/ratebroker/limiter/ring)
- Heap (https://github.com/parkerroan/ratebroker/limiter/heap)
*/
package ratebroker
