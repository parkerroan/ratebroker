package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/redis/go-redis/v9"

	"github.com/gorilla/mux"
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

	limiter := redis_rate.NewLimiter(rdb)

	// Create a new router
	r := mux.NewRouter() // or http.NewServeMux()

	// Add the logging middleware first.
	r.Use(LoggingMiddleware)

	// Create a new rate limited HTTP handler using your middleware
	r.Use(RedisRateLimitMiddleware(limiter))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Handle the request, i.e., serve content, call other functions, etc.
		w.Write([]byte("Hello, World!"))
	})

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

func RedisRateLimitMiddleware(limiter *redis_rate.Limiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userKey := r.Header.Get("X-User-ID")
			if userKey == "" {
				userKey = r.RemoteAddr
			}

			res, err := limiter.Allow(r.Context(), userKey, redis_rate.PerSecond(10))
			if err != nil {
				slog.Error("Error checking rate limit", slog.Any("error", err.Error()))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			h := w.Header()
			h.Set("RateLimit-Remaining", strconv.Itoa(res.Remaining))

			if res.Allowed == 0 {
				// We are rate limited.

				seconds := int(res.RetryAfter / time.Second)
				h.Set("RateLimit-RetryAfter", strconv.Itoa(seconds))
				w.WriteHeader(http.StatusTooManyRequests)
				// Stop processing and return the error.
				return
			}

			// Continue to the next middleware or handler.
			next.ServeHTTP(w, r)
		})
	}
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
