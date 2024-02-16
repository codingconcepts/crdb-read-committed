package database

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MustConnect makes a connection to the database and fails if one
// can't be established.
func MustConnect(url string, concurrency int) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		log.Fatalf("error parsing connection string: %v", err)
	}

	// Open enough connections for both readers and writers.
	cfg.MaxConns = int32(concurrency) * 2

	db, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}

	return db
}
