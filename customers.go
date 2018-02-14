package main

import (
	"database/sql"
	"fmt"
)

type Customer struct {
	ID          string `json:"id"`
	FullName    string `json:"full_name"`
	CompanyName string `json:"company_name"`
	Email       string `json:"email"`
	Address     string `json:"address"`
	PostalCode  string `json:"postal_code"`
	City        string `json:"city"`
	Country     string `json:"country"`
}

func getCustomers(db *sql.DB) ([]Customer, error) {
	return queryCustomers(db, nil, nil, nil)
}

func getProductCustomers(db *sql.DB, productId, limit int) ([]Customer, error) {
	return queryCustomers(db, nil, &productId, &limit)
}

func getCustomer(db *sql.DB, id int) (*Customer, error) {
	customers, err := queryCustomers(db, &id, nil, nil)
	if err != nil || len(customers) == 0 {
		return nil, err
	}
	return &customers[0], nil
}

func queryCustomers(db *sql.DB, id, productId, limit *int) ([]Customer, error) {
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

	rows, err := db.Query(queryString, args...)
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
