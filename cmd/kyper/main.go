package main

import (
	"os"

	"github.com/bitfootco/kyper-cli/internal/cmd"
	"github.com/bitfootco/kyper-cli/internal/ui"
)

func main() {
	if err := cmd.Execute(); err != nil {
		ui.PrintError(err.Error())
		os.Exit(1)
	}
}
