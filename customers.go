package main

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Customer struct {
	ID          int    `json:"id"`
	FullName    string `json:"full_name"`
	CompanyName string `json:"company_name"`
	Email       string `json:"email"`
	Address     string `json:"address"`
	PostalCode  string `json:"postal_code"`
	City        string `json:"city"`
	Country     string `json:"country"`
}

func getCustomers(ctx context.Context, db *sqlx.DB) ([]Customer, error) {
	return queryCustomers(ctx, db, nil, nil, nil)
}

func getProductCustomers(ctx context.Context, db *sqlx.DB, productId, limit int) ([]Customer, error) {
	return queryCustomers(ctx, db, nil, &productId, &limit)
}

func getCustomer(ctx context.Context, db *sqlx.DB, id int) (*Customer, error) {
	customers, err := queryCustomers(ctx, db, &id, nil, nil)
	if err != nil || len(customers) == 0 {
		return nil, err
	}
	return &customers[0], nil
}

func queryCustomers(ctx context.Context, db *sqlx.DB, id, productId, limit *int) ([]Customer, error) {
	var args []interface{}
	queryString := `
SELECT
  customers.id, full_name, company_name, email,
  address, postal_code, city, country
FROM customers
`
	if id != nil {
		queryString += "WHERE id=?\n"
		args = append(args, *id)
	}
	if productId != nil {
		queryString += "" +
			"JOIN orders ON customers.id=orders.customer_id " +
			"JOIN order_lines ON orders.id=order_lines.order_id " +
			"WHERE order_lines.product_id=?\n"
		args = append(args, *productId)
	}
	if limit != nil {
		queryString += fmt.Sprintf("LIMIT %d\n", *limit)
	}

	rows, err := db.QueryContext(ctx, db.Rebind(queryString), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(
			&c.ID, &c.FullName, &c.CompanyName,
			&c.Email, &c.Address, &c.PostalCode,
			&c.City, &c.Country,
		); err != nil {
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}
