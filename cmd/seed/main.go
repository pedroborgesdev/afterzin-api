package main

import (
	"os"
	"path/filepath"

	"afterzin/api/internal/config"
	"afterzin/api/internal/db"
	"afterzin/api/internal/db/seeds"
	"afterzin/api/internal/logger"
)

func main() {
	cfg := config.Load()

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0755); err != nil {
		logger.Fatalf("erro ao criar diretório de dados: %v", err)
	}

	sqlite, err := db.OpenSQLite(cfg.DBPath)
	if err != nil {
		logger.Fatalf("erro ao abrir banco de dados: %v", err)
	}
	defer sqlite.Close()

	if err := db.Migrate(sqlite); err != nil {
		logger.Fatalf("erro ao executar migrações: %v", err)
	}

	logger.Infof("executando seeds...")
	if err := seeds.Run(sqlite); err != nil {
		logger.Fatalf("erro ao executar seeds: %v", err)
	}
	logger.Infof("seeds finalizados com sucesso")
}
