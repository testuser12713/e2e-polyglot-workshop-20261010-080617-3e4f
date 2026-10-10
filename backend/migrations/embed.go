// Package migrations embeds the SQL migration files so the binary carries its
// own schema and never depends on a file path in the working tree.
package migrations

import "embed"

// FS holds every *.sql migration in this directory, applied in filename order.
//
//go:embed *.sql
var FS embed.FS
