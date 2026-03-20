package migrations

import "embed"

// FS embeds all SQL migration files for use in the Go binary.
//
//go:embed *.sql
var FS embed.FS
