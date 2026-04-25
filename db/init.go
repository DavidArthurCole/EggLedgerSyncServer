package db

import (
	"database/sql"

	"github.com/pkg/errors"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var _db *sql.DB

func Init(connStr string) error {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return errors.Wrap(err, "db.Init: open")
	}
	if err := db.Ping(); err != nil {
		return errors.Wrap(err, "db.Init: ping")
	}
	_db = db
	return runMigrations()
}

func Close() {
	if _db != nil {
		_db.Close()
	}
}

func DB() *sql.DB { return _db }
