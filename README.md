# Rate Broker [![Go Reference](https://pkg.go.dev/badge/github.com/parkerroan/ratebroker.svg)](https://pkg.go.dev/github.com/parkerroan/ratebroker)

# RateBroker

RateBroker is a rate limiting library written in Go. It uses a Limiter to enforce rate limits and a MessageBroker to allow for distributed limiting. 

The library also provides a way to configure the RateBroker with various options, such as setting the maximum number of requests, the time window for the rate limit, and the NTP server for time synchronization.

## Features

- Configurable rate limiting options
- NTP server support for accurate time synchronization
- Message broker integration for distributed rate limiting
- In-memory caching for efficient rate limit tracking
- This repo also provides the [Limiter](https://github.com/yourusername/yourproject/tree/main/limiter) subpackage for rate limiting functionality

## Distributed Use Case

The use of a message broker can allow for rate limiting across application/servers. Since it is a message broker, it is not atomic rate limiting but should be conisidered more of approximate rate limiting. 

The advantage of this approach is that the actual rate limiting happens in the server memory/application versus requiring an external network call like traditional solutions. 

This means that checking the rate of users is very fast and in the instance of blocking users, not even a publish call is made over the network (refer to sequence diagram). Therefore; 429 use cases require very little latency to shut down in the scenario of DOS attacks. 

Rate Broker is best used in server side applications versus client side applications. Although it certainly can be used in both scenarios, in the case of client side the benefit is not as impactful.

### Request Sequence

![Request Sequence](https://substackcdn.com/image/fetch/w_1456,c_limit,f_webp,q_auto:good,fl_progressive:steep/https%3A%2F%2Fsubstack-post-media.s3.amazonaws.com%2Fpublic%2Fimages%2F2c78d46f-5827-4a43-b34b-7424b9b00c86_2288x1879.png)

### Example Diagram w/ Redis Streams

![Redis Distributed Arch](https://substackcdn.com/image/fetch/w_1456,c_limit,f_webp,q_auto:good,fl_progressive:steep/https%3A%2F%2Fsubstack-post-media.s3.amazonaws.com%2Fpublic%2Fimages%2Fa60ff4eb-fe92-4e7e-9432-a3ede2516573_1290x1003.png)

## Usage

### Non Distributed Example

First, import the `ratebroker` package in your Go code:

```go
import "github.com/parkerroan/ratebroker"
```

Then, create a new RateBroker with the desired options:

```go
// create a rate broker with max rate of 100 req / min
rb := ratebroker.NewRateBroker(
    ratebroker.WithMaxRequests(100),
    ratebroker.WithWindow(1 * time.Minute),
)
```

You can then use the `TryAccept` method to check if a new request should be accepted based on the current rate limit:

```go
ok, details := rb.TryAccept(context.Background(), "user1")
if !ok {
    log.Printf("Rate limit exceeded. Max requests: %d, Window: %s", details.MaxRequests, details.Window)
} else {
    log.Println("Request accepted")
}
```

### Distributed HTTP Server Example

```go
import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/parkerroan/ratebroker"

	"github.com/gorilla/mux"
	"golang.org/x/exp/slog"
)

func main() {
    rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Create instances of your broker and limiter
	redisBroker := ratebroker.NewRedisMessageBroker(rdb)

	// Create a rate broker w/ ring limiter
	rateBroker := ratebroker.NewRateBroker(
		ratebroker.WithBroker(redisBroker),
		ratebroker.WithMaxRequests(cfg.MaxRequests),
		ratebroker.WithWindow(cfg.Window),
	)

	ctx := context.Background()
	rateBroker.Start(ctx)

	// This function generates a key (in this case, the client's IP address)
	// that the rate limiter uses to identify unique clients.
	keyGetter := func(r *http.Request) string {
		// You might want to improve this method to handle IP-forwarding, etc.
		return r.RemoteAddress
	}

	// Create a new router
	r := mux.NewRouter() // or http.NewServeMux()

	// Create a new rate limited HTTP handler using your middleware
	r.Use(ratebroker.HttpMiddleware(rateBroker, keyGetter))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Handle the request, i.e., serve content, call other functions, etc.
		w.Write([]byte("Hello, World!"))
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}

```

## Contributing

Contributions are welcome. Please submit a pull request or create an issue to discuss the changes.

## License

This project is licensed under the MIT License.