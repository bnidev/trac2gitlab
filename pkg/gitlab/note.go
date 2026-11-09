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

// ListIssueNotes lists all notes on a project issue, following pagination.
func (c *Client) ListIssueNotes(projectID any, issueIID int64) ([]*Note, error) {
	var all []*Note
	page := int64(1)
	for {
		notes, resp, err := c.git.Notes.ListIssueNotes(projectID, issueIID, &gitlab_client.ListIssueNotesOptions{
			ListOptions: gitlab_client.ListOptions{PerPage: 100, Page: page},
		})
		if err != nil {
			return nil, err
		}
		all = append(all, notes...)
		if resp == nil || resp.NextPage == 0 {
			return all, nil
		}
		page = int64(resp.NextPage)
	}
}
