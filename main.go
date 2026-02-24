package main

import (
	"os"

	"github.com/gi4nks/tecla/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
