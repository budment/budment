package main

import (
	"os"

	"github.com/budment/budment/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
