package state

import (
	"io/fs"

	"github.com/tzone85/project-x/migrations"
)

func testMigrationsFS() fs.FS {
	return migrations.FS
}
