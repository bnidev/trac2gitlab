package gitlab

import (
	gitlab_client "gitlab.com/gitlab-org/api/client-go"
)

// Issue represents a GitLab issue, here it is aliased to the GitLab client type for easier usage.
type Issue = gitlab_client.Issue

// GetIssue retrieves a specific issue by its ID from the specified project.
func (c *Client) GetIssue(projectID any, issueID int) (*Issue, error) {
	issue, _, err := c.git.Issues.GetIssue(projectID, issueID)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

// CreateIssueOptions defines the options for creating a new issue in GitLab, here it is aliased to the GitLab client type for easier usage.
type CreateIssueOptions = gitlab_client.CreateIssueOptions

// CreateIssue creates a new issue in the specified project with the provided options.
func (c *Client) CreateIssue(projectID any, opts *gitlab_client.CreateIssueOptions) (*Issue, error) {
	issue, _, err := c.git.Issues.CreateIssue(projectID, opts)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

// UpdateIssueOptions defines the options for updating an existing issue in GitLab, here it is aliased to the GitLab client type for easier usage.
type UpdateIssueOptions = gitlab_client.UpdateIssueOptions

// UpdateIssue updates an existing issue in the specified project with the provided options.
func (c *Client) UpdateIssue(projectID any, issueID int, opts *gitlab_client.UpdateIssueOptions) (*Issue, error) {
	issue, _, err := c.git.Issues.UpdateIssue(projectID, issueID, opts)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

// ListProjectIssues retrieves a list of issues for the specified project.
func (c *Client) ListProjectIssues(projectID any) ([]*Issue, error) {
	issues, _, err := c.git.Issues.ListProjectIssues(projectID, &gitlab_client.ListProjectIssuesOptions{})
	if err != nil {
		return nil, err
	}
	return issues, nil
}
