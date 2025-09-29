package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kal997/radius-accounting-system/internal/config"
	"github.com/kal997/radius-accounting-system/internal/models"
	"github.com/kal997/radius-accounting-system/internal/storage"

	"github.com/joho/godotenv"

	"layeh.com/radius"
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

	// Initialize storage
	store, err := storage.NewRedisStorage(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Test storage connection
	if err := store.HealthCheck(context.Background()); err != nil {
		log.Fatalf("Storage health check failed: %v", err)
	}

	log.Printf("Starting RADIUS accounting server on %s", cfg.GetRADIUSAddr())
	log.Printf("Connected to Redis at %s", cfg.GetRedisAddr())

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping RADIUS server...")
		cancel()
	}()

	// Start RADIUS server
	server := radius.PacketServer{
		Handler:      radius.HandlerFunc(handleAccounting(store)),
		SecretSource: radius.StaticSecretSource([]byte(cfg.GetSharedSecret())),
		Addr:         cfg.GetRADIUSAddr(),
		Network:      "udp",
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		log.Println("Shutting down...")
	case err := <-serverErr:
		if err != nil {
			log.Fatalf("RADIUS server failed: %v", err)
		}
	}
}

func handleAccounting(store storage.Storage) func(w radius.ResponseWriter, r *radius.Request) {
	return func(w radius.ResponseWriter, r *radius.Request) {

		var responseCode radius.Code = radius.CodeAccountingResponse

		if r.Code != radius.CodeAccountingRequest {
			log.Printf("Received non-accounting request: %d", r.Code)

		} else {

			clientIP := getClientIP(r)

			record, err := models.NewAccountingRecordFromRADIUS(r.Packet, clientIP)
			if err != nil {
				log.Printf("Failed to parse accounting packet: %v", err)
			} else if err := record.Validate(); err != nil {
				log.Printf("Invalid accounting record: %v", err)
			} else if err := store.Store(context.Background(), record); err != nil {
				log.Printf("Failed to store accounting record: %v", err)
			} else {
				log.Printf("Stored accounting record: %s", record.GenerateRedisKey())
			}
		}

		// Send accounting response following RADUIS specs
		response := r.Response(responseCode)
		if err := w.Write(response); err != nil {
			log.Printf("Failed to send response: %v", err)
		}
	}
}
func getClientIP(r *radius.Request) string {
	if addr, ok := r.RemoteAddr.(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return r.RemoteAddr.String()
}
