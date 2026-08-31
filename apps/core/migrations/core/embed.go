// Package core holds the embedded SQL migration files for the core app.
//
// It exists so the database package can embed the migrations without relying on
// a "../.." path (which //go:embed forbids). The embedded filesystem is rooted
// at this directory, so consumers should pass "." as the base path to iofs.
package core

import "embed"

// FS contains all *.sql migration files in this directory.
//
//go:embed *.sql
var FS embed.FS
