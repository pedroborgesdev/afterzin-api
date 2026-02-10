package db

import (
	"database/sql"
	"embed"
	"io/fs"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY);`); err != nil {
		return err
	}
	var current int
	_ = db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&current)

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(e.Name(), "_", 2)
		if len(parts) != 2 {
			continue
		}
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		if version <= current {
			continue
		}
		body, err := fs.ReadFile(migrationsFS, "migrations/"+e.Name())
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(body)); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO schema_version (version) VALUES (?)`, version); err != nil {
			return err
		}
		current = version
	}
	return nil
}
