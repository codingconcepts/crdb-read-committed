package database

import (
	"context"
	"log"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MustConnect makes a connection to the database and fails if one
// can't be established.
func MustConnect(url string) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		log.Fatalf("error parsing connection string: %v", err)
	}
	cfg.MaxConns = 100

	db, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("error connecting to db: %v", err)
	}

	return db
}

// Product returns a random product name and price.
func Product() (string, float64) {
	return gofakeit.ProductName(), float64(int(gofakeit.Float64Range(1, 100)*100)) / 100
}
