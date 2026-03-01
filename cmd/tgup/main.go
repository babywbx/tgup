package main

import (
	"os"

	"github.com/babywbx/tgup/internal/cli"
)

// Set via ldflags at build time.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cli.SetVersion(version, commit, date)
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
