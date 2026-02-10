package graphql

import (
	"database/sql"

	"afterzin/api/internal/config"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	DB     *sql.DB
	Config *config.Config
}
