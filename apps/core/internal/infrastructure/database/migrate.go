package database

import (
	"context"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	coremigrations "github.com/nofendian17/sbterm/apps/core/migrations/core"
	// postgres driver: registers the "postgres://" URL scheme via its init().
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// migrationsFS holds the embedded core migration SQL files.
var migrationsFS = coremigrations.FS

// RunMigrations applies the embedded core migrations (users, rbac, watchlists)
// against the database identified by dbURL. It is idempotent: running it again
// after a successful apply returns nil (migrate.ErrNoChange is ignored).
func RunMigrations(ctx context.Context, dbURL string) error {
	src, err := iofs.New(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("migrations: source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return fmt.Errorf("migrations: new: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrations: up: %w", err)
	}
	return nil
}
