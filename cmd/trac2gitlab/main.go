package main

import (
	"log/slog"

	"github.com/bnidev/trac2gitlab/internal/cli"
	"github.com/bnidev/trac2gitlab/internal/utils"
)

func main() {
	cli.Execute()
	logger := utils.NewLogger("")
	slog.SetDefault(logger.Logger)
}
