package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/kal997/radius-accounting-system/internal/config"
	"github.com/kal997/radius-accounting-system/internal/logger"
	"github.com/kal997/radius-accounting-system/internal/notifier"
)

func main() {

	if value, ok := os.LookupEnv("ENV"); ok && value == "prod" {
		// In Docker/Compose, rely only on provided env vars
	} else {
		// Local dev: force load .env
		if err := godotenv.Overload(); err != nil {
			log.Fatalf("Could not load .env: %v", err)
		}
	}
	// Load configuration into config
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize notifier, worst case 5s before timeout
	redis, err := notifier.NewRedisNotifier(cfg.GetRedisAddr())
	if err != nil {
		log.Fatalf("Failed to initialize notifier: %v", err)
	}
	defer redis.Close()

	// Test notifier connection
	if err := redis.HealthCheck(context.Background()); err != nil {
		log.Fatalf("Notifier health check failed: %v", err)
	}

	// Initialize file logger
	fileLogger, err := logger.NewFileLogger(cfg.GetLogFile())
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer fileLogger.Close()

	log.Printf("Starting radius-controlplane-logger")
	log.Printf("Connected to Redis at %s", cfg.GetRedisAddr())
	log.Printf("Logging to file: %s", cfg.GetLogFile())

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping...")
		cancel()
	}()

	// Subscribe to Redis keyspace notifications
	events, err := redis.Subscribe(ctx, []string{"radius:acct:*"})
	if err != nil {
		log.Fatalf("Failed to subscribe to notifications: %v", err)
	}

	log.Println("Listening for Redis keyspace notifications...")

	// Process events
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		case event, ok := <-events:
			if !ok {
				log.Println("Event channel closed")
				return
			}

			// Log all operations
			message := fmt.Sprintf("Received update for key: %s, Operation: %s", event.Key, event.Operation)
			if err := fileLogger.Log(ctx, message); err != nil {
				log.Printf("Failed to log event: %v", err)
			} else if cfg.IsDebugEnabled() {
				log.Printf("Logged: %s", message)
			}

		}
	}
}
