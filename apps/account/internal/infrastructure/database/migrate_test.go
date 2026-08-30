//go:build integration

package database

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunMigrations applies the embedded migrations against a real Postgres.
// Set ACCOUNT_DB_URL to a postgres:// DSN to enable. Skips otherwise.
func TestRunMigrations(t *testing.T) {
	dbURL := os.Getenv("ACCOUNT_DB_URL")
	if dbURL == "" {
		t.Skip("ACCOUNT_DB_URL not set; skipping integration migration test")
	}

	ctx := context.Background()
	err := RunMigrations(ctx, dbURL)
	require.NoError(t, err)

	// Re-running must be idempotent (ErrNoChange is swallowed).
	err = RunMigrations(ctx, dbURL)
	require.NoError(t, err)
}
