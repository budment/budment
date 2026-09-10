package main

import (
	"os"

	"github.com/budment/budment/internal/cli"
)

var (
	version = "0.1.0-alpha.0"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetVersionInfo(version, commit, date)
	os.Exit(cli.Execute())
}
