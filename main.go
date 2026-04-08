package main

import (
	"log/slog"
	"os"

	"github.com/ganasa18/go-template/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		slog.Error("fatal error", slog.Any("error", err))
		os.Exit(1)
	}
}
