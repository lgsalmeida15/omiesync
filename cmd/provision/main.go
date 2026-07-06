package main

import (
	"context"
	"fmt"
	"os"

	"omie-sync-api/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: provision <schema_name>")
		os.Exit(1)
	}
	schema := os.Args[1]

	pool, err := db.NewPool(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "db: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	p := db.NewProvisioner(pool)
	if err := p.ProvisionSchema(context.Background(), schema); err != nil {
		fmt.Fprintf(os.Stderr, "provision: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Schema '%s' provisionado com sucesso\n", schema)
}
