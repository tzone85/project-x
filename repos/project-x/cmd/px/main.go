package main

import (
	"github.com/tzone85/project-x/internal/cli"
)

var version = "dev"

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
