package main

import (
	"os"

	"github.com/tzone85/project-x/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
