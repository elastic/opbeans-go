package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type Order struct {
	ID           int                `json:"id"`
	CreatedAt    time.Time          `json:"created_at"`
	CustomerID   int                `json:"customer_id"`
	CustomerName string             `json:"customer_name,omitempty"`
	Lines        []ProductOrderLine `json:"lines,omitempty"`
}

type ProductOrderLine struct {
	Product
	Amount int `json:"amount"`
}

func getOrders(ctx context.Context, db *sql.DB) ([]Order, error) {
	const limit = 1000
	queryString := `SELECT
  orders.id, orders.created_at,
  customers.id, customers.full_name
FROM orders JOIN customers ON orders.customer_id=customers.id
`
	queryString += fmt.Sprintf("LIMIT %d\n", limit)

	rows, err := db.QueryContext(ctx, queryString)
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

func getOrder(ctx context.Context, db *sql.DB, id int) (*Order, error) {
	queryString := `SELECT
  orders.id, orders.created_at, customer_id
FROM orders WHERE orders.id=?
`
	row := db.QueryRowContext(ctx, queryString, id)
	var order Order
	if err := row.Scan(&order.ID, &order.CreatedAt, &order.CustomerID); err != nil {
		return nil, errors.Wrap(err, "querying order")
	}

	// NOTE(axw) order lines aren't rendered by the UI, but are kept here for
	// consistency with opbeans(-python). Its presence will have an impact on
	// latency at least.
	queryString = `SELECT
  product_id, amount,
  products.sku, products.name, products.description,
  products.type_id, products.stock, products.cost, products.selling_price
FROM products JOIN order_lines ON products.id=order_lines.product_id
WHERE order_lines.order_id=?
`
	rows, err := db.QueryContext(ctx, queryString, id)
	if err != nil {
		return nil, errors.Wrap(err, "querying product order lines")
	}
	defer rows.Close()

	var lines []ProductOrderLine
	for rows.Next() {
		var l ProductOrderLine
		if err := rows.Scan(
			&l.ID, &l.Amount,
			&l.SKU, &l.Name, &l.Description,
			&l.TypeID, &l.Stock, &l.Cost, &l.SellingPrice,
		); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	order.Lines = lines
	return &order, rows.Err()
}
