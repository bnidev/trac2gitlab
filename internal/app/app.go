package app

import (
	"log/slog"

	"github.com/bnidev/trac2gitlab/internal/config"
)

var AppVersion = "0.0.0"

type AppContext struct {
	Logger *slog.Logger
	Config *config.Config
}
