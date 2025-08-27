package main

import (
	"log/slog"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/cli"
	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/utils"
)

func main() {
	defaultLogLevel := "info"
	logger := utils.NewLogger(defaultLogLevel)
	slog.SetDefault(logger.Logger)

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load configuration", "errorMsg", err)
		return
	}

	if cfg.General.LogLevel != "" {
		logger.SetLevel(cfg.General.LogLevel)
	}

	ctx := &app.AppContext{
		Config: &cfg,
		Logger: logger.Logger,
	}

	cli.Execute(ctx)
}
