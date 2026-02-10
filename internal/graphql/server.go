package graphql

import (
	"database/sql"
	"embed"
	"io/fs"
	"net/http"

	"afterzin/api/internal/config"
	"github.com/99designs/gqlgen/graphql/handler"
	gqlparser "github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

//go:embed schema/*.graphqls
var schemaFS embed.FS

func NewHandler(db *sql.DB, cfg *config.Config) http.Handler {
	schema, err := loadSchema()
	if err != nil {
		panic("load schema: " + err.Error())
	}
	resolver := &Resolver{DB: db, Config: cfg}
	es := NewExecutableSchema(Config{
		Schema:    schema,
		Resolvers: resolver,
	})
	return handler.NewDefaultServer(es)
}

func loadSchema() (*ast.Schema, error) {
	body, err := fs.ReadFile(schemaFS, "schema/schema.graphqls")
	if err != nil {
		return nil, err
	}
	return gqlparser.LoadSchema(&ast.Source{Input: string(body), Name: "schema.graphqls"})
}
