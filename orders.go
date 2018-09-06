package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
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

func getOrders(ctx context.Context, db *sqlx.DB) ([]Order, error) {
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

func getOrder(ctx context.Context, db *sqlx.DB, id int) (*Order, error) {
	queryString := db.Rebind(`SELECT
  orders.id, orders.created_at, customer_id
FROM orders WHERE orders.id=?`)

	row := db.QueryRowContext(ctx, queryString, id)
	var order Order
	if err := row.Scan(&order.ID, &order.CreatedAt, &order.CustomerID); err != nil {
		return nil, errors.Wrap(err, "querying order")
	}

	queryString = db.Rebind(`SELECT
  product_id, amount,
  products.sku, products.name, products.description,
  products.type_id, products.stock, products.cost, products.selling_price
FROM products JOIN order_lines ON products.id=order_lines.product_id
WHERE order_lines.order_id=?`)

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

func createOrder(ctx context.Context, db *sqlx.DB, customer *Customer, lines []ProductOrderLine) (int, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	driver := db.DriverName()
	returningID := "RETURNING id"
	if driver == "sqlite3" {
		returningID = ""
	}
	insertOrderStmt := db.Rebind("INSERT INTO orders (customer_id) VALUES (?) " + returningID)

	insertOrderLineStmt, err := tx.PrepareContext(ctx, db.Rebind(
		"INSERT INTO order_lines (order_id, product_id, amount) VALUES(?, ?, ?)",
	))
	if err != nil {
		return -1, errors.Wrap(err, "failed to prepare insert order lines statement")
	}
	defer insertOrderLineStmt.Close()

	var orderID int
	if returningID == "" {
		result, err := tx.ExecContext(ctx, insertOrderStmt, customer.ID)
		if err != nil {
			return -1, err
		}
		rowID, err := result.LastInsertId()
		if err != nil {
			return -1, err
		}
		orderID = int(rowID)
	} else {
		err := tx.QueryRowContext(ctx, insertOrderStmt, customer.ID).Scan(&orderID)
		if err != nil {
			return -1, err
		}
	}
	for _, line := range lines {
		if _, err := insertOrderLineStmt.ExecContext(ctx, orderID, line.Product.ID, line.Amount); err != nil {
			return -1, err
		}
	}
	if err := tx.Commit(); err != nil {
		return -1, err
	}
	return orderID, nil
}
