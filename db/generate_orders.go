package opbeansdb

import (
	"context"
	"math/rand"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	maxOrderAmount = 3
)

// GenerateOrders generates n orders, randomizing the
// products and customers in use.
func GenerateOrders(db *sqlx.DB, driver string, n int, rng *rand.Rand) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	productIDs, err := getIDs(ctx, db, "products")
	if err != nil {
		return err
	}

	customerIDs, err := getIDs(ctx, db, "customers")
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	returningID := "RETURNING id"
	if driver == "sqlite3" {
		returningID = ""
	}

	insertOrderStmt, err := tx.PrepareContext(ctx, db.Rebind(
		"INSERT INTO orders (customer_id) VALUES (?) "+returningID,
	))
	if err != nil {
		return errors.Wrap(err, "failed to prepare insert orders statement")
	}
	defer insertOrderStmt.Close()

	insertOrderLineStmt, err := tx.PrepareContext(ctx, db.Rebind(
		"INSERT INTO order_lines (order_id, product_id, amount) VALUES(?, ?, ?)",
	))
	if err != nil {
		return errors.Wrap(err, "failed to prepare insert order lines statement")
	}
	defer insertOrderLineStmt.Close()

	for i := 0; i < n; i++ {
		productID := productIDs[rng.Intn(len(productIDs))]
		customerID := customerIDs[rng.Intn(len(customerIDs))]
		var orderID int64
		if driver == "sqlite3" {
			result, err := insertOrderStmt.ExecContext(ctx, customerID)
			if err != nil {
				return err
			}
			rowID, err := result.LastInsertId()
			if err != nil {
				return err
			}
			orderID = rowID
		} else {
			err := insertOrderStmt.QueryRowContext(ctx, customerID).Scan(&orderID)
			if err != nil {
				return err
			}
		}

		amount := rng.Intn(maxOrderAmount + 1)
		if _, err := insertOrderLineStmt.ExecContext(ctx, orderID, productID, amount); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func getIDs(ctx context.Context, db *sqlx.DB, table string) ([]int, error) {
	var ids []int
	rows, err := db.QueryContext(ctx, "SELECT id FROM "+table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Close()
}
