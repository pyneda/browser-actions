package main

import (
	"context"
	"os"

	"github.com/pyneda/browser-actions/internal/cli"
)

func main() {
	os.Exit(cli.Execute(context.Background(), os.Args[1:]))
}
