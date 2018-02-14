package main

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

type Product struct {
	SKU          string `json:"sku"`
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Stock        int    `json:"stock"`
	Cost         int    `json:"cost"`
	SellingPrice int    `json:"selling_price"`
	Sold         int    `json:"sold,omitempty"`
	TypeID       int    `json:"type_id,omitempty"`
	TypeName     string `json:"type_name,omitempty"`
}

type ProductType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func getProducts(db *sql.DB) ([]Product, error) {
	return queryProducts(db, nil)
}

func getTopProducts(db *sql.DB) ([]Product, error) {
	const limit = 3 // top 3 best-selling products
	queryString := `SELECT
	  id, sku, name, stock, SUM(order_lines.amount) AS sold
FROM products JOIN order_lines ON id=product_id GROUP BY product_id ORDER BY sold DESC
`
	queryString += fmt.Sprintf("LIMIT %d\n", limit)

	rows, err := db.Query(queryString)
	if err != nil {
		return nil, errors.Wrap(err, "querying top products")
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Stock, &p.Sold); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func getProduct(db *sql.DB, id int) (*Product, error) {
	products, err := queryProducts(db, &id)
	if err != nil || len(products) == 0 {
		return nil, err
	}
	return &products[0], nil
}

func queryProducts(db *sql.DB, id *int) ([]Product, error) {
	var args []interface{}
	queryString := `SELECT
  products.id, products.sku, products.name, products.description,
  products.stock, products.cost, products.selling_price,
  products.type_id, product_types.name
FROM products JOIN product_types ON type_id=product_types.id
`
	if id != nil {
		queryString += "WHERE products.id=?\n"
		args = append(args, *id)
	}

	rows, err := db.Query(queryString, args...)
	if err != nil {
		return nil, errors.Wrap(err, "querying products")
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.ID, &p.SKU, &p.Name, &p.Description,
			&p.Stock, &p.Cost, &p.SellingPrice,
			&p.TypeID, &p.TypeName,
		); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func getProductTypes(db *sql.DB) ([]ProductType, error) {
	rows, err := db.Query("SELECT id, name FROM product_types")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var productTypes []ProductType
	for rows.Next() {
		var pt ProductType
		if err := rows.Scan(&pt.ID, &pt.Name); err != nil {
			return nil, err
		}
		productTypes = append(productTypes, pt)
	}
	return productTypes, rows.Err()
}
