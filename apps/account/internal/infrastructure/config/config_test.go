package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.account.yaml")
	require.NoError(t, os.WriteFile(p, []byte("app:\n  name: account\n  version: dev\nport: \":8081\"\nauth:\n  jwt_secret: test-secret\n  access_ttl: 15m\n  refresh_ttl: 720h\n  default_user_ttl: 720h\n  bcrypt_cost: 12\n"), 0o644))

	// Load reads the config from the working directory ("."), so run it from
	// the temp dir where the file was written.
	old, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(old) })

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, ":8081", cfg.Port)
	require.Equal(t, "test-secret", cfg.Auth.JWTSecret)
	require.Equal(t, 12, cfg.Auth.BcryptCost)
}
