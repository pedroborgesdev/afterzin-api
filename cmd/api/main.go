package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"afterzin/api/internal/config"
	"afterzin/api/internal/db"
	"afterzin/api/internal/graphql"
	"afterzin/api/internal/middleware"
	"afterzin/api/internal/pagarme"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists (ignores error if file is absent)
	_ = godotenv.Load()

	cfg := config.Load()

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	sqlite, err := db.OpenSQLite(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer sqlite.Close()

	if err := db.Migrate(sqlite); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	graphqlHandler := graphql.NewHandler(sqlite, cfg)

	// Build HTTP mux with all routes
	mux := http.NewServeMux()
	mux.Handle("/graphql", graphqlHandler)

	// Pagar.me REST endpoints (only registered when PAGARME_API_KEY is set)
	if cfg.PagarmeAPIKey != "" {
		pagarmeClient := pagarme.NewClient(
			cfg.PagarmeAPIKey,
			cfg.PagarmeWebhookSecret,
			cfg.PagarmeRecipientID,
			cfg.PagarmeAppFee,
			cfg.BaseURL,
		)
		pagarmeHandler := pagarme.NewHandler(pagarmeClient, sqlite, cfg)
		mux.HandleFunc("/v1/pagarme/recipient/create", pagarmeHandler.CreateRecipient)
		mux.HandleFunc("/v1/pagarme/recipient/status", pagarmeHandler.GetRecipientStatus)
		mux.HandleFunc("/v1/pagarme/payment/create", pagarmeHandler.CreatePayment)
		mux.HandleFunc("/v1/pagarme/payment/status", pagarmeHandler.GetPaymentStatus)
		mux.HandleFunc("/v1/pagarme/webhook", pagarmeHandler.HandleWebhook)
		log.Println("Pagar.me endpoints registered (Recipient + PIX Payment + Webhook)")
	} else {
		log.Println("PAGARME_API_KEY not set — Pagar.me endpoints disabled")
	}

	handler := middleware.CORS(cfg.CORSOrigins)(middleware.Auth(cfg.JWTSecret)(mux))

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("GraphQL server listening on %s", addr)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("server stopped")
}
