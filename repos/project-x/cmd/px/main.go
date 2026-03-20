package main

import (
	"fmt"
	"os"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("px %s (commit: %s, built: %s)\n", version, commit, date)
		return
	}
	fmt.Println("px - AI Agent Orchestration CLI")
	fmt.Println("Run 'px version' for version info")
}
