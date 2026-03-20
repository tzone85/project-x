// Package migrations embeds all SQL migration files so they ship
// inside the compiled binary. No external files needed at runtime.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
