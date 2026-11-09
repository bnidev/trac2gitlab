package main

import (
	"log/slog"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/cli"
	"github.com/bnidev/trac2gitlab/internal/utils"
)

func main() {
	logger := utils.NewLogger("info")
	slog.SetDefault(logger.Logger)

	ctx := &app.AppContext{Logger: logger.Logger}
	cli.Execute(ctx)
}
