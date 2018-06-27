package main

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type Stats struct {
	Products  int `json:"products"`
	Customers int `json:"customers"`
	Orders    int `json:"orders"`
	Numbers   struct {
		Revenue int `json:"revenue"`
		Cost    int `json:"cost"`
		Profit  int `json:"profit"`
	} `json:"numbers"`
}

func getStats(ctx context.Context, db *sqlx.DB) (*Stats, error) {
	var stats Stats
	countParams := []struct {
		table  string
		result *int
	}{
		{"products", &stats.Products},
		{"customers", &stats.Customers},
		{"orders", &stats.Orders},
	}
	for _, p := range countParams {
		row := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+p.table)
		if err := row.Scan(p.result); err != nil {
			return nil, errors.Wrap(err, "querying "+p.table)
		}
	}

	var revenue, cost, profit *int
	row := db.QueryRowContext(ctx, `
SELECT
  SUM(selling_price), SUM(cost), SUM(selling_price-cost)
FROM products JOIN order_lines ON products.id=order_lines.product_id
`)
	if err := row.Scan(&revenue, &cost, &profit); err != nil {
		return nil, errors.Wrap(err, "querying numbers")
	}
	stats.Numbers.Revenue = maybeInt(revenue)
	stats.Numbers.Cost = maybeInt(cost)
	stats.Numbers.Profit = maybeInt(profit)
	return &stats, nil
}

func maybeInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
