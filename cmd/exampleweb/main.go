package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/parkerroan/ratebroker"
	"github.com/parkerroan/ratebroker/limiter"

	"golang.org/x/exp/slog"
)

type Config struct {
	Port        int           `envconfig:"SERVER_PORT" default:"8080"`
	MaxRequests int           `envconfig:"MAX_REQUESTS" default:"5"`
	Window      time.Duration `envconfig:"WINDOW_DURATION" default:"60s"` // This can be time.Duration if you want the library to handle parsing
	RedisURL    string        `envconfig:"REDIS_URL" default:"localhost:6379"`
}

func main() {
	// Load .env file from given path. We're assuming it's in the current directory.
	// Don't forget to check for errors.
	loadEnvFile()

	// Initialize components of your application here
	// For example, create a new ratebroker, set up routes, etc.
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL, // "localhost:6379"
	})

	// Create instances of your broker and limiter
	redisBroker := ratebroker.NewRedisMessageBroker(rdb)

	// Create a rate broker w/ ring limiter
	rateBroker := ratebroker.NewRateBroker(
		ratebroker.WithLimiterContructorFunc(limiter.NewRingLimiterConstructorFunc()),
		ratebroker.WithBroker(redisBroker),
		ratebroker.WithMaxRequests(cfg.MaxRequests),
		ratebroker.WithWindow(cfg.Window),
	)

	ctx := context.Background()
	rateBroker.Start(ctx)

	// This function generates a key (in this case, the client's IP address)
	// that the rate limiter uses to identify unique clients.
	keyGetter := func(r *http.Request) string {
		// Get a custom header X-User-ID from the request
		// If it doesn't exist, use the remote address
		userKey := r.Header.Get("X-User-ID")
		if userKey == "" {
			userKey = r.RemoteAddr
		}
		return userKey
	}

	// Use the default mux as the router for pprof
	// r := http.NewServeMux()

	// Create a new router with mux
	r := mux.NewRouter()

	// Add the logging middleware for gorilla/mux
	r.Use(LoggingMiddleware)

	// Middleware to rate limit requests for gorilla/mux
	r.Use(ratebroker.HttpMiddleware(rateBroker, keyGetter))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Handle the request, i.e., serve content, call other functions, etc.
		w.Write([]byte("Hello, World!"))
	})

	// // Add the logging middleware first for net/http
	// wrappedHandler := LoggingMiddleware(r)

	// // Create a new rate limited HTTP handler using your middleware for net/http
	// wrappedHandler = ratebroker.HttpMiddleware(rateBroker, keyGetter)(r)

	// log.Fatal(http.ListenAndServe(":8080", wrappedHandler)) // net/http
	log.Fatal(http.ListenAndServe(":8080", r))
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and writes it to the response.
func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new status recorder.
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default to 200 OK if WriteHeader is not called.
		}

		// Continue to the next middleware or handler.
		next.ServeHTTP(recorder, r)

		// Now that the handler has finished, the status code is set.
		log.Printf(
			"Method: %s | Path: %s | StatusCode: %d | RemoteAddr: %s | UserAgent: %s",
			r.Method,
			r.RequestURI,
			recorder.statusCode,
			r.RemoteAddr,
			r.UserAgent(),
		)
	})
}

func loadEnvFile() {
	if _, err := os.Stat(".env"); err == nil {
		// The file exists, now let's try to load it
		if err := godotenv.Load(); err != nil {
			// The file couldn't be loaded, log the error
			log.Fatalf("Error loading .env file: %s", err)
		}
	} else if !os.IsNotExist(err) {
		// There's an error other than "file does not exist", let's log it
		slog.Warn(fmt.Sprintf("Unexpected error looking for .env file: %s", err))
	}
}
