package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type Order struct {
	ID           int       `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	CustomerID   int       `json:"customer_id"`
	CustomerName string    `json:"customer_name,omitempty"`
}

func getOrders(db *sql.DB) ([]Order, error) {
	limit := 1000
	return queryOrders(db, nil, &limit)
}

func getOrder(db *sql.DB, id int) (*Order, error) {
	orders, err := queryOrders(db, &id, nil)
	if err != nil || len(orders) == 0 {
		return nil, err
	}
	return &orders[0], nil
}

func queryOrders(db *sql.DB, id *int, limit *int) ([]Order, error) {
	var args []interface{}
	queryString := `SELECT
  orders.id, orders.created_at,
  customers.id, customers.full_name
FROM orders JOIN customers ON orders.customer_id=customers.id
`
	if id != nil {
		queryString += "WHERE orders.id=?\n"
		args = append(args, *id)
	}
	if limit != nil {
		queryString += fmt.Sprintf("LIMIT %d\n", *limit)
	}

	rows, err := db.Query(queryString, args...)
	if err != nil {
		return nil, errors.Wrap(err, "querying orders")
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(
			&o.ID, &o.CreatedAt,
			&o.CustomerID, &o.CustomerName,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}
