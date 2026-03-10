package main

import (
	"os"

	"github.com/tomohiro-owada/affine-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
