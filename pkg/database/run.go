package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FetchIDs returns a selection of ids from the account table.
func FetchIDs(db *pgxpool.Pool, count int) ([]string, error) {
	const stmt = `SELECT id FROM account ORDER BY random() LIMIT $1`

	rows, err := db.Query(context.Background(), stmt, count)
	if err != nil {
		return nil, fmt.Errorf("querying for rows: %w", err)
	}

	var accountIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning id: %w", err)
		}
		accountIDs = append(accountIDs, id)
	}

	return accountIDs, nil
}

// FetchBalance fetches the balance for a given account id.
func FetchBalance(ctx context.Context, tx pgx.Tx, id string) (float64, error) {
	const stmt = `SELECT balance FROM account WHERE id = $1`

	row := tx.QueryRow(ctx, stmt, id)

	var balance float64
	err := row.Scan(&balance)
	return balance, err
}

// UpdateBalance sets the balance for an account.
func UpdateBalance(ctx context.Context, tx pgx.Tx, id string, newBalance float64) error {
	const stmt = `UPDATE account SET balance = $1 WHERE id = $2`

	_, err := tx.Exec(ctx, stmt, newBalance, id)
	return err
}

// FetchBalancesSum returns the sum of all balances in the account table.
func FetchBalancesSum(db *pgxpool.Pool, ids []string) (float64, error) {
	const stmt = `SELECT SUM(balance) FROM account WHERE id = ANY($1)`

	row := db.QueryRow(context.Background(), stmt, ids)

	var sum float64
	err := row.Scan(&sum)
	return sum, err
}
