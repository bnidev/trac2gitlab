package app

import (
	"fmt"
	"log/slog"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
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
	if a.Config.GitLab.ProjectID > 0 && a.Config.GitLab.ProjectPath != "" {
		return nil
	}
	if a.Config.GitLab.Project == "" && a.Config.GitLab.ProjectID <= 0 {
		return fmt.Errorf("gitlab.project_id or gitlab.project is required")
	}

	client, err := gitlab.NewGitLabClient(a.Config)
	if err != nil {
		return err
	}
	ref := any(a.Config.GitLab.Project)
	if a.Config.GitLab.Project == "" {
		ref = a.Config.GitLab.ProjectID
	}
	project, err := client.GetProject(ref)
	if err != nil {
		return fmt.Errorf("resolve gitlab project %q: %w", a.Config.GitLab.Project, err)
	}
	a.Config.GitLab.ProjectID = int(project.ID)
	a.Config.GitLab.ProjectPath = project.PathWithNamespace
	return nil
}
