# Limiter

The "limiter" subpackage provides rate limiting functionality with two different implementations: `RingLimiter` and `HeapLimiter`.

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