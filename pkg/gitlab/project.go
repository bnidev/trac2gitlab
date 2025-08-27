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

// GetProjectMembers retrieves the members of a specific project by its ID.
func (c *Client) GetProjectMembers(projectID any) ([]*gitlab.ProjectMember, error) {
	members, _, err := c.git.ProjectMembers.ListAllProjectMembers(projectID, &gitlab.ListProjectMembersOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get members for project %v: %w", projectID, err)
	}

	return members, nil
}

// GetProjectMember retrieves a specific member of a project by project ID and user ID.
func (c *Client) GetProjectMember(projectID any, userID int) (*gitlab.ProjectMember, error) {
	member, _, err := c.git.ProjectMembers.GetProjectMember(projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member %d for project %v: %w", userID, projectID, err)
	}

	return member, nil
}

// SetProjectMemberAccessLevel updates a project member's access level.
func (c *Client) SetProjectMemberAccessLevel(projectID any, userID int, accessLevel gitlab.AccessLevelValue) error {
	if !ValidateAccessLevel(int(accessLevel)) {
		return fmt.Errorf("invalid access level: %d", accessLevel)
	}

	_, _, err := c.git.ProjectMembers.EditProjectMember(projectID, userID, &gitlab.EditProjectMemberOptions{
		AccessLevel: &accessLevel,
	})
	if err != nil {
		return fmt.Errorf("failed to set access level for user %d in project %v: %w", userID, projectID, err)
	}

	return nil
}

// AddProjectMember adds a user to a project with a specified access level.
func (c *Client) AddProjectMember(projectID any, userID int, accessLevel gitlab.AccessLevelValue) error {
	if !ValidateAccessLevel(int(accessLevel)) {
		return fmt.Errorf("invalid access level: %d", accessLevel)
	}

	_, _, err := c.git.ProjectMembers.AddProjectMember(projectID, &gitlab.AddProjectMemberOptions{
		UserID:      &userID,
		AccessLevel: &accessLevel,
	})
	if err != nil {
		return fmt.Errorf("failed to add user %d to project %v: %w", userID, projectID, err)
	}

	return nil
}

// ValidateAccessLevel checks if the provided access level is valid.
// Valid levels are:
// No access (0)
// Minimal access (5)
// Guest (10)
// Planner (15)
// Reporter (20)
// Developer (30)
// Maintainer (40)
// Owner (50).
func ValidateAccessLevel(level int) bool {
	switch level {
	case 0, 5, 10, 15, 20, 30, 40, 50:
		return true
	default:
		return false
	}
}
