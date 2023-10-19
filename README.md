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

## Usage

First, import the `ratebroker` package in your Go code:

```go
import "github.com/parkerroan/ratebroker"
```

Then, create a new RateBroker with the desired options:

```go
rb := ratebroker.NewRateBroker(
    ratebroker.WithMaxRequests(100),
    ratebroker.WithWindow(1 * time.Minute),
    ratebroker.WithNTPServer("pool.ntp.org"),
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

## Contributing

Contributions are welcome. Please submit a pull request or create an issue to discuss the changes.

## License

This project is licensed under the MIT License.