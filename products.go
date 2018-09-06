package main

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
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

func getProducts(ctx context.Context, db *sqlx.DB) ([]Product, error) {
	return queryProducts(ctx, db, nil)
}

func getTopProducts(ctx context.Context, db *sqlx.DB) ([]Product, error) {
	const limit = 3 // top 3 best-selling products
	queryString := `SELECT
	  id, sku, name, stock, SUM(order_lines.amount) AS sold
FROM products JOIN order_lines ON id=product_id GROUP BY products.id ORDER BY sold DESC
`
	queryString += fmt.Sprintf("LIMIT %d\n", limit)

	rows, err := db.QueryContext(ctx, queryString)
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

func getProduct(ctx context.Context, db *sqlx.DB, id int) (*Product, error) {
	products, err := queryProducts(ctx, db, &id)
	if err != nil || len(products) == 0 {
		return nil, err
	}
	return &products[0], nil
}

func queryProducts(ctx context.Context, db *sqlx.DB, id *int) ([]Product, error) {
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

	rows, err := db.QueryContext(ctx, db.Rebind(queryString), args...)
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

func getProductTypes(ctx context.Context, db *sqlx.DB) ([]ProductType, error) {
	return queryProductTypes(ctx, db, nil)
}

func getProductType(ctx context.Context, db *sqlx.DB, id int) (*ProductType, error) {
	productTypes, err := queryProductTypes(ctx, db, &id)
	if err != nil || len(productTypes) == 0 {
		return nil, err
	}
	return &productTypes[0], nil
}

func queryProductTypes(ctx context.Context, db *sqlx.DB, id *int) ([]ProductType, error) {
	var args []interface{}
	queryString := "SELECT id, name FROM product_types"
	if id != nil {
		queryString += " WHERE id=?"
		args = append(args, *id)
	}

	rows, err := db.QueryContext(ctx, db.Rebind(queryString), args...)
	if err != nil {
		return nil, errors.Wrap(err, "querying product types")
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
