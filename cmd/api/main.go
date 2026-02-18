package main

import (
	"afterzin/api/internal/logger"
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
		logger.Fatalf("erro ao criar diretório de dados: %v", err)
	}

	sqlite, err := db.OpenSQLite(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer sqlite.Close()

	if err := db.Migrate(sqlite); err != nil {
		logger.Fatalf("erro ao executar migrações: %v", err)
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
		mux.HandleFunc("/v1/recipient/create", pagarmeHandler.CreateRecipient)
		mux.HandleFunc("/v1/recipient/status", pagarmeHandler.GetRecipientStatus)
		mux.HandleFunc("/v1/payment/create", pagarmeHandler.CreatePayment)
		mux.HandleFunc("/v1/payment/status", pagarmeHandler.GetPaymentStatus)
		mux.HandleFunc("/v1/webhook", pagarmeHandler.HandleWebhook)
		logger.Infof("endpoints do Pagar.me registrados (Recipient + PIX + Webhook)")
	} else {
		logger.Warnf("PAGARME_API_KEY não definido — endpoints do Pagar.me desabilitados")
	}

	handler := middleware.CORS(cfg.CORSOrigins)(middleware.Auth(cfg.JWTSecret)(mux))

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	logger.Infof("servidor GraphQL escutando em %s", addr)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("erro no servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Infof("encerrando...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatalf("erro ao encerrar servidor: %v", err)
	}
	logger.Infof("servidor parado")
}
