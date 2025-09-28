package main

import (
	"context"
	"log"
	"net"

	"github.com/joho/godotenv"
	"github.com/kal997/radius-accounting-system/internal/config"
	"github.com/kal997/radius-accounting-system/internal/models"
	"github.com/kal997/radius-accounting-system/internal/storage"

	"layeh.com/radius"
)

func main() {

	// Force load and overwrite existing env vars
	if err := godotenv.Overload(); err != nil { // Use Overload instead of Load
		log.Printf("Warning: Error loading .env file: %v", err)
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

	// Start RADIUS server
	server := radius.PacketServer{
		Handler:      radius.HandlerFunc(handleAccounting(store)),
		SecretSource: radius.StaticSecretSource([]byte(cfg.GetSharedSecret())),
		Addr:         cfg.GetRADIUSAddr(), // ":1813" for accounting
		Network:      "udp",
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("RADIUS server failed: %v", err)
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
