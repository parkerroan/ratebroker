# Limiter

The "limiter" subpackage provides rate limiting functionality with two different implementations: `RingLimiter` and `HeapLimiter`.

`HeapLimiter` sorts on insertion based on the value (timestamp) while `RingLimiter` assumed order based on insertion. So this would be used when you insertions may be asynchronous and not in order and you need every bit of accuracy. Because of the extra operations, heap is more expensive. In most cases `RingLimiter` will be faster and sufficient. 

Both methods are very fast, here is benchmark comparing them: 

```shell
goos: darwin
goarch: arm64
pkg: github.com/parkerroan/ratebroker/limiter
BenchmarkRingLimiter-10    	29388708	        40.32 ns/op	      24 B/op	       1 allocs/op
BenchmarkHeapLimiter-10    	15745207	        71.66 ns/op	      80 B/op	       1 allocs/op
```

## Features

- Two rate limiting strategies: Ring Buffer and Min Heap.
- Thread-safe operations using mutex locks.
- Customizable rate limiting parameters: size and window.

## Usage

First, import the `limiter` package in your Go code:

```go
import "github.com/yourusername/yourproject/limiter"
```

Then, create a new `RingLimiter` or `HeapLimiter` with the desired size and window:

```go
rl := limiter.NewRingLimiter(100, 1 * time.Minute)
hl := limiter.NewHeapLimiter(100, 1 * time.Minute)
```

You can then use the `Try`, `Accept`, and `TryAccept` methods to enforce rate limits:

```go
ok := rl.TryAccept(time.Now())
```

## Contributing

Contributions are welcome. Please submit a pull request or create an issue to discuss the changes.

## License

This project is licensed under the MIT License.