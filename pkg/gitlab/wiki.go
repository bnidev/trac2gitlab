package gitlab

import (
	"io"

	gitlab_client "gitlab.com/gitlab-org/api/client-go"
)

// Wiki represents a GitLab wiki page.
type Wiki = gitlab_client.Wiki

// WikiAttachment represents a GitLab wiki attachment.
type WikiAttachment = gitlab_client.WikiAttachment

// UploadWikiAttachmentOptions defines the options for uploading a wiki attachment.
type UploadWikiAttachmentOptions = gitlab_client.UploadWikiAttachmentOptions

// WikiFormatMarkdown is the GitLab wiki Markdown format.
const WikiFormatMarkdown = gitlab_client.WikiFormatMarkdown

// CreateWikiPageOptions defines the options for creating a wiki page.
type CreateWikiPageOptions = gitlab_client.CreateWikiPageOptions

// EditWikiPageOptions defines the options for editing a wiki page.
type EditWikiPageOptions = gitlab_client.EditWikiPageOptions

// ListWikisOptions defines the options for listing wiki pages.
type ListWikisOptions = gitlab_client.ListWikisOptions

// ListWikis lists project wiki pages.
func (c *Client) ListWikis(projectID any, withContent bool) ([]*Wiki, error) {
	pages, _, err := c.git.Wikis.ListWikis(projectID, &gitlab_client.ListWikisOptions{WithContent: &withContent})
	if err != nil {
		return nil, err
	}
	return pages, nil
}

// CreateWikiPage creates a project wiki page.
func (c *Client) CreateWikiPage(projectID any, opts *gitlab_client.CreateWikiPageOptions) (*Wiki, error) {
	page, _, err := c.git.Wikis.CreateWikiPage(projectID, opts)
	if err != nil {
		return nil, err
	}
	return page, nil
}

// EditWikiPage edits a project wiki page by slug.
func (c *Client) EditWikiPage(projectID any, slug string, opts *gitlab_client.EditWikiPageOptions) (*Wiki, error) {
	page, _, err := c.git.Wikis.EditWikiPage(projectID, slug, opts)
	if err != nil {
		return nil, err
	}
	return page, nil
}

// UploadWikiAttachment uploads an attachment to a project wiki.
func (c *Client) UploadWikiAttachment(projectID any, content io.Reader, filename string, opts *gitlab_client.UploadWikiAttachmentOptions) (*WikiAttachment, error) {
	attachment, _, err := c.git.Wikis.UploadWikiAttachment(projectID, content, filename, opts)
	if err != nil {
		return nil, err
	}
	return attachment, nil
}
