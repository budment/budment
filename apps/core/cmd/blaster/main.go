package main

import (
	"os"

	"github.com/vunas/blaster/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
