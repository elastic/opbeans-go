package opbeansdb

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/pkg/errors"
)

const (
	maxOrderAmount = 3
)

// GenerateOrders generates n orders, randomizing the
// products and customers in use.
func GenerateOrders(db *sql.DB, driver string, n int, rng *rand.Rand) error {
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

	arg0, arg1, arg2 := "$1", "$2", "$3"
	returningID := "RETURNING id"
	if driver == "sqlite3" {
		arg0, arg1, arg2 = "?", "?", "?"
		returningID = ""
	}

	insertOrderStmt, err := tx.PrepareContext(ctx, fmt.Sprintf(
		"INSERT INTO orders (customer_id) VALUES (%s) %s",
		arg0, returningID,
	))
	if err != nil {
		return errors.Wrap(err, "failed to prepare insert orders statement")
	}
	defer insertOrderStmt.Close()

	insertOrderLineStmt, err := tx.PrepareContext(ctx, fmt.Sprintf(
		"INSERT INTO order_lines (order_id, product_id, amount) VALUES(%s, %s, %s)",
		arg0, arg1, arg2,
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

func getIDs(ctx context.Context, db *sql.DB, table string) ([]int, error) {
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
