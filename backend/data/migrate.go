package data

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/hashicorp/go-hclog"
)

// RunMigrations applies all pending database migrations from the given directory.
func RunMigrations(dsn string, migrationsPath string) error {
	logger := hclog.Default()

	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	version, dirty, _ := m.Version()
	logger.Info("Migrations", "version", version, "dirty", dirty)
	return nil
}
