package gitlab

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GetProjectList retrieves a list of projects from GitLab and prints their IDs and names.
func (c *Client) GetProjectList() error {
	projects, _, err := c.git.Projects.ListProjects(&gitlab.ListProjectsOptions{})
	if err != nil {
		return err
	}
	for _, project := range projects {
		fmt.Printf("Project ID: %d, Name: %s\n", project.ID, project.Name)
	}
	return nil
}

// Project represents a GitLab project with essential fields.
type Project struct {
	ID                int    `json:"id"`
	Description       string `json:"description"`
	DefaultBranch     string `json:"default_branch"`
	Name              string `json:"name"`
	NameWithNamespace string `json:"name_with_namespace"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
}

// GetProject retrieves a specific project by its ID.
func (c *Client) GetProject(id any) (*gitlab.Project, error) {
	project, _, err := c.git.Projects.GetProject(id, &gitlab.GetProjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get project %d: %w", id, err)
	}
	return project, nil
}
