package main

import (
	"os"

	"github.com/fuzzspec/fuzzspec/cmd/fuzzspec/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
