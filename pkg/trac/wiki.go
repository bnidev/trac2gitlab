package trac

import (
	"fmt"
	"time"
	"trac2gitlab/internal/utils"
)

// GetWikiPageNames retrieves the names of all wiki pages
func (c *Client) GetWikiPageNames() ([]string, error) {
	var result []string
	err := c.rpc.Call("wiki.getAllPages", nil, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// WikiPage represents metadata for a wiki page
type WikiPage struct {
    Author       string     `json:"author"`
    Comment      *string    `json:"comment"`
    LastModified time.Time  `json:"lastModified"`
    Name         string     `json:"name"`
    Version      int64        `json:"version"`
		Attachments  []Attachment `json:"attachments,omitempty"`
}

// GetWikiPage retrieves the content of a wiki page
func (c *Client) GetWikiPage(pageName string, version int) (*string, error) {
	var content string
	err := c.rpc.Call("wiki.getPage", pageName, &content)
	if err != nil {
		return nil, err
	}

	content = utils.TracToMarkdown(content)

	return &content, nil
}

// GetWikiPageInfo retrieves metadata for a wiki page
func (c *Client) GetWikiPageInfo(pageName string) (*WikiPage, error) {
	var raw map[string]any
	err := c.rpc.Call("wiki.getPageInfo", pageName, &raw)
	if err != nil {
		return nil, err
	}
	return c.decodeWikiPage(raw)
}

// GetWikiPageInfoVersion retrieves metadata for a specific version of a wiki page
func (c *Client) GetWikiPageInfoVersion(pageName string, version int64) (*WikiPage, error) {
	var raw map[string]any
	err := c.rpc.Call("wiki.getPageInfo", []any{pageName, version}, &raw)
	if err != nil {
		return nil, err
	}
	return c.decodeWikiPage(raw)
}

// GetWikiPageVersion retrieves a specific version of a wiki page
func (c *Client) GetWikiPageVersion(pageName string, version int64) (*string, error) {
	var content string
	err := c.rpc.Call("wiki.getPage", []any{pageName, version}, &content)
	if err != nil {
		return nil, err
	}

	content = utils.TracToMarkdown(content)

	return &content, nil
}

// decodeWikiPage converts raw map data from the RPC response into a WikiPage struct
func (c *Client) decodeWikiPage(raw map[string]any) (*WikiPage, error) {
	page := &WikiPage{
		Author:  utils.GetString(raw["author"]),
		Name:    utils.GetString(raw["name"]),
		Version: utils.GetInt64(raw["version"]),
	}

	if comment, ok := raw["comment"].(string); ok {
		page.Comment = &comment
	}

	if lm, ok := raw["lastModified"].(time.Time); ok {
		page.LastModified = lm
	}

	attachments, err := ListAttachments(c, ResourceWiki, page.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments for ticket %s: %w", page.Name, err)
	}

	page.Attachments = attachments

	return page, nil
}
