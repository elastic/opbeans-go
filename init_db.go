package main

import (
	"context"
	"math/rand"
	"time"

	opbeansdb "github.com/elastic/opbeans-go/db"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func initDatabase(db *sqlx.DB, driver string) error {
	if orders, err := getOrders(context.Background(), db); err == nil {
		if len(orders) != 0 {
			return nil
		}
	}

	filenames := []string{
		"schema_" + driver + ".sql",
		"customers.sql",
		"products.sql",
	}
	logrus.Infof("initializing %q database", driver)
	for _, filename := range filenames {
		logrus.Infof("executing %q", filename)
		f, err := opbeansdb.SQL.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := opbeansdb.ExecCommands(context.Background(), db, f); err != nil {
			return errors.Wrapf(err, "executing %q", filename)
		}
	}

	const numOrders = 5000
	logrus.Infof("generating %d random orders", numOrders)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return opbeansdb.GenerateOrders(db, driver, numOrders, rng)
}
