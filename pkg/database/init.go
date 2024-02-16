package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Create the database infrastructure required by the load test.
func Create(db *pgxpool.Pool) error {
	const stmt = `CREATE TABLE account (
									id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
									balance DECIMAL NOT NULL
								)`

	_, err := db.Exec(context.Background(), stmt)
	return err
}

// Seed the database with data.
func Seed(db *pgxpool.Pool, rowCount int) error {
	const stmt = `INSERT INTO account (balance)
								SELECT 10000
								FROM generate_series(1, $1)`

	_, err := db.Exec(context.Background(), stmt, rowCount)
	return err
}

// Drop the database infrastructure required by the load test.
func Drop(db *pgxpool.Pool) error {
	const stmt = `DROP TABLE IF EXISTS account`

	_, err := db.Exec(context.Background(), stmt)
	return err
}
