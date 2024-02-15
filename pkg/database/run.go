package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FetchIDs returns all of the ids in the product table.
func FetchIDs(db *pgxpool.Pool) ([]string, error) {
	const stmt = `SELECT id FROM product`

	rows, err := db.Query(context.Background(), stmt)
	if err != nil {
		return nil, fmt.Errorf("querying for rows: %w", err)
	}

	var productIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scanning id: %w", err)
		}
		productIDs = append(productIDs, id)
	}

	return productIDs, nil
}
