package gitlab

import (
	"fmt"
	"log/slog"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GetProjectList retrieves a list of projects from GitLab and prints their IDs and names.
func (c *Client) GetProjectList() error {
	projects, _, err := c.git.Projects.ListProjects(&gitlab.ListProjectsOptions{})
	if err != nil {
		return err
	}
	for _, project := range projects {

		slog.Debug("Project found", "ID", project.ID, "Name", project.Name)
	}
	return nil
}

// GetProject retrieves a specific project by its ID.
func (c *Client) GetProject(id any) (*gitlab.Project, error) {
	project, _, err := c.git.Projects.GetProject(id, &gitlab.GetProjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get project %v: %w", id, err)
	}
	return project, nil
}
