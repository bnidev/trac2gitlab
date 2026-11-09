package gitlab

import (
	gitlab_client "gitlab.com/gitlab-org/api/client-go"
)

// Note represents a GitLab note.
type Note = gitlab_client.Note

// CreateIssueNoteOptions defines the options for creating an issue note.
type CreateIssueNoteOptions = gitlab_client.CreateIssueNoteOptions

// CreateIssueNote creates a note on a project issue.
func (c *Client) CreateIssueNote(projectID any, issueIID int64, opts *gitlab_client.CreateIssueNoteOptions) (*Note, error) {
	note, _, err := c.git.Notes.CreateIssueNote(projectID, issueIID, opts)
	if err != nil {
		return nil, err
	}
	return note, nil
}
