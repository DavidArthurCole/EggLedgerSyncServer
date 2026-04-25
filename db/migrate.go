package db

import (
	"embed"
	"fmt"

	"github.com/pkg/errors"
)

//go:embed migrations/*.sql
var _migrations embed.FS

func runMigrations() error {
	// Create schema_version table if missing
	_, err := _db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`)
	if err != nil {
		return errors.Wrap(err, "runMigrations: create schema_version")
	}
	var version int
	row := _db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`)
	row.Scan(&version)

	migrations := []string{"1_initial_schema.up.sql"}
	for i, name := range migrations {
		if i+1 <= version {
			continue
		}
		content, err := _migrations.ReadFile("migrations/" + name)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("runMigrations: read %s", name))
		}
		if _, err := _db.Exec(string(content)); err != nil {
			return errors.Wrap(err, fmt.Sprintf("runMigrations: exec %s", name))
		}
		if _, err := _db.Exec(`INSERT INTO schema_version (version) VALUES ($1)`, i+1); err != nil {
			return errors.Wrap(err, fmt.Sprintf("runMigrations: update version to %d", i+1))
		}
	}
	return nil
}
