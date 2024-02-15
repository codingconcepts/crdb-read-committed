package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Create the database infrastructure required by the load test.
func Create(db *pgxpool.Pool) error {
	const stmt = `CREATE TABLE product (
									id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
									name STRING NOT NULL,
									price DECIMAL NOT NULL
								)`

	_, err := db.Exec(context.Background(), stmt)
	return err
}

// Seed the database with data.
func Seed(db *pgxpool.Pool, rowCount int) error {
	const stmt = `INSERT INTO product (id, name, price) VALUES ($1, $2, $3)`

	batch := pgx.Batch{}
	for i := 0; i < rowCount; i++ {
		id := uuid.NewString()
		name, price := Product()
		batch.Queue(stmt, id, name, price)
	}

	if _, err := db.SendBatch(context.Background(), &batch).Exec(); err != nil {
		return fmt.Errorf("running insert job: %w", err)
	}

	return nil
}
