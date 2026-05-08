package sqlite

import (
	"database/sql"
	"errors"

	"github.com/bak1an/artf/store"
	sqlite "modernc.org/sqlite"
	sqlitelib "modernc.org/sqlite/lib"
)

// mapSQLErr translates database/sql and modernc.org/sqlite errors into the
// store package's sentinel errors. Any unrecognized error is returned as-is.
func mapSQLErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return store.ErrNotFound
	}
	var se *sqlite.Error
	if errors.As(err, &se) {
		switch se.Code() {
		case sqlitelib.SQLITE_CONSTRAINT_UNIQUE, sqlitelib.SQLITE_CONSTRAINT_PRIMARYKEY:
			return store.ErrAlreadyExists
		}
	}
	return err
}
