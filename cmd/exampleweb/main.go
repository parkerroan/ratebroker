package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/parkerroan/ratebroker"
	"github.com/parkerroan/ratebroker/broker"
	"github.com/parkerroan/ratebroker/limiter"

	"golang.org/x/exp/slog"
)

type Config struct {
	Port        int    `envconfig:"SERVER_PORT" default:"8080"`
	MaxRequests int    `envconfig:"MAX_REQUESTS" default:"100"`
	Window      string `envconfig:"WINDOW_DURATION" default:"1m"` // This can be time.Duration if you want the library to handle parsing
	RedisURL    string `envconfig:"REDIS_URL" default:"redis://localhost:6379"`
	// ... other configuration variables
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

	// Create instances of your broker and limiter
	redisBroker := broker.NewRedisBroker(cfg.RedisURL)

	// Create a rate broker
	rateBroker := ratebroker.NewRateBroker(redisBroker, limiter.NewRingLimiter, ratebroker.WithMaxRequests(30))

	rateBroker.Start(context.Background())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Use your rate broker to check or enforce rate limits
		if !rateBroker.TryAccept("test-user") {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		// Handle the request, i.e., serve content, call other functions, etc.
		w.Write([]byte("Hello, World!"))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
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
