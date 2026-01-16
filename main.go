package main

import (
	"os"

	"github.com/bluefunda/cai-cli/cmd/ai"
)

func main() {
	if err := ai.Execute(); err != nil {
		os.Exit(1)
	}
}
