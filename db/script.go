package opbeansdb

import (
	"bufio"
	"bytes"
	"context"
	"io"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func ExecCommands(ctx context.Context, db *sqlx.DB, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Split(scanStatements)
	for scanner.Scan() {
		stmt := scanner.Text()
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return errors.Wrapf(err, "executing statement failed:\n%s", stmt)
		}
	}
	return nil
}

func scanStatements(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, ';'); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return 0, nil, errors.New("unterminated statement")
	}
	// Request more data.
	return 0, nil, nil
}
