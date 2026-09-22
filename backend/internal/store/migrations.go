package store

import "embed"

// MigrationsFS holds the numbered DDL files. Each version is a pair
// NNNN_name.up.sql / NNNN_name.down.sql; Migrate applies the .up.sql files in
// filename order. Never edit a migration that has been applied — add a new one.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
