package main

import (
	"log"
	"os"
	"path/filepath"

	"afterzin/api/internal/config"
	"afterzin/api/internal/db"
	"afterzin/api/internal/db/seeds"
)

func main() {
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

	log.Println("Running seeds...")
	if err := seeds.Run(sqlite); err != nil {
		log.Fatalf("seeds: %v", err)
	}
	log.Println("Seeds completed successfully.")
}
