package gitlab

import (
	"io"

	gitlab_client "gitlab.com/gitlab-org/api/client-go"
)

// ProjectMarkdownUploadedFile represents an uploaded project file.
type ProjectMarkdownUploadedFile = gitlab_client.ProjectMarkdownUploadedFile

// UploadProjectMarkdown uploads a file to the project's markdown uploads.
func (c *Client) UploadProjectMarkdown(projectID any, content io.Reader, filename string) (*ProjectMarkdownUploadedFile, error) {
	file, _, err := c.git.ProjectMarkdownUploads.UploadProjectMarkdown(projectID, content, filename)
	if err != nil {
		return nil, err
	}
	return file, nil
}
