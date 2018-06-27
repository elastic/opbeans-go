package opbeansdb

import (
	"context"
	"math/rand"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateOrdersSQLite3(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	requireExecCommands(t, db, "sql/schema_sqlite3.sql")
	requireExecCommands(t, db, "sql/products.sql")
	requireExecCommands(t, db, "sql/customers.sql")
	assertGenerateOrders(t, db, "sqlite3")
}

func TestGenerateOrdersPostgres(t *testing.T) {
	if os.Getenv("PGHOST") == "" {
		t.Skip("PGHOST not set")
	}
	db, err := sqlx.Open("postgres", "")
	require.NoError(t, err)
	defer db.Close()

	db.Exec("CREATE DATABASE opbeans_go_test")
	defer db.Exec("DROP DATABASE opbeans_go_test")

	requireExecCommands(t, db, "sql/schema_postgres.sql")
	requireExecCommands(t, db, "sql/products.sql")
	requireExecCommands(t, db, "sql/customers.sql")
	assertGenerateOrders(t, db, "postgres")
}

func assertGenerateOrders(t *testing.T, db *sqlx.DB, driver string) {
	rng := rand.New(rand.NewSource(0))
	err := GenerateOrders(db, driver, 100, rng)
	if !assert.NoError(t, err) {
		return
	}

	var ordersCount int
	row := db.QueryRow("SELECT COUNT(*) FROM orders")
	err = row.Scan(&ordersCount)
	require.NoError(t, err)
	assert.Equal(t, 100, ordersCount)

	var orderLinesCount int
	row = db.QueryRow("SELECT COUNT(*) FROM order_lines")
	err = row.Scan(&orderLinesCount)
	require.NoError(t, err)
	assert.Equal(t, 100, orderLinesCount)
}

func requireExecCommands(t *testing.T, db *sqlx.DB, filename string) {
	f, err := os.Open(filename)
	require.NoError(t, err)
	defer f.Close()
	err = ExecCommands(context.Background(), db, f)
	require.NoError(t, err)
}
