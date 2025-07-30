package gitlab

import (
	"gitlab.com/gitlab-org/api/client-go"
)

type Issue = gitlab.Issue

func (c *Client) GetIssue(projectID any, issueID int) (*Issue, error) {
	issue, _, err := c.git.Issues.GetIssue(projectID, issueID)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

type CreateIssueOptions = gitlab.CreateIssueOptions

func (c *Client) CreateIssue(projectID any, opts *gitlab.CreateIssueOptions) (*Issue, error) {
	issue, _, err := c.git.Issues.CreateIssue(projectID, opts)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

type UpdateIssueOptions = gitlab.UpdateIssueOptions

func (c *Client) UpdateIssue(projectID any, issueID int, opts *gitlab.UpdateIssueOptions) (*Issue, error) {
	issue, _, err := c.git.Issues.UpdateIssue(projectID, issueID, opts)
	if err != nil {
		return nil, err
	}
	return issue, nil
}


func (c *Client) ListProjectIssues(projectID any) ([]*Issue, error) {
	issues, _, err := c.git.Issues.ListProjectIssues(projectID, &gitlab.ListProjectIssuesOptions{})
	if err != nil {
		return nil, err
	}
	return issues, nil
}
