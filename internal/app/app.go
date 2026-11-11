package app

import (
	"fmt"
	"log/slog"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/pipeline"
)

var AppVersion = "0.0.0"

type AppContext struct {
	Logger     *slog.Logger
	Config     *config.Config
	ProjectDir string
}

// EnsureProjectID resolves gitlab.project to a numeric ID and path when needed.
func (a *AppContext) EnsureProjectID() error {
	if a.Config == nil {
		return fmt.Errorf("configuration not loaded")
	}
	return pipeline.ResolveProjectID(a.Config)
}
