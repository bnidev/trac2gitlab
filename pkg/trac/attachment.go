package trac

import (
	"encoding/base64"
	"fmt"
	"time"
	"trac2gitlab/internal/utils"
)

// Attachment represents a file attachment in Trac
type Attachment struct {
	Filename    string
	Description string
	Size        int64
	Time        time.Time
	Author      string
}

// ResourceType is either "ticket" or "wiki"
type ResourceType string

const (
	ResourceTicket ResourceType = "ticket"
	ResourceWiki   ResourceType = "wiki"
)

// ListAttachments retrieves all attachments for a given resource type and ID
func ListAttachments(c *Client, resType ResourceType, id any) ([]Attachment, error) {
	var method string
	switch resType {
	case ResourceTicket:
		method = "ticket.listAttachments"
	case ResourceWiki:
		method = "wiki.listAttachments"
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resType)
	}

	var raw []any
	err := c.rpc.Call(method, []any{id}, &raw)
	if err != nil {
		return nil, err
	}

	attachments := make([]Attachment, 0, len(raw))
	for i, item := range raw {
		tuple, ok := item.([]any)
		if !ok || len(tuple) != 5 {
			fmt.Printf("skipping: unexpected format at index %d: %#v\n", i, item)
			continue
		}

		filename := fmt.Sprintf("%v", tuple[0])
		description := ""
		if tuple[1] != nil {
			description = fmt.Sprintf("%v", tuple[1])
		}

		size, ok := utils.ToInt64(tuple[2])
		if !ok {
			fmt.Printf("skipping: unexpected size type at index %d: %#v\n", i, tuple[2])
			continue
		}

		parsedTime, err := utils.ParseRequiredTracTime(tuple[3])
		if err != nil {
			fmt.Printf("skipping: failed to parse time at index %d: %v\n", i, err)
			continue
		}

		author := fmt.Sprintf("%v", tuple[4])

		attachments = append(attachments, Attachment{
			Filename:    filename,
			Description: description,
			Size:        size,
			Time:        parsedTime,
			Author:      author,
		})
	}

	return attachments, nil
}

// GetAttachment downloads an attachment by its resource type and identifiers
func GetAttachment(c *Client, resType ResourceType, id any, filename string) ([]byte, error) {
	var method string
	var args []any

	switch resType {
	case ResourceTicket:
		method = "ticket.getAttachment"
		args = []any{id, filename}
	case ResourceWiki:
		method = "wiki.getAttachment"
		args = []any{fmt.Sprintf("%s/%s", id, filename)}
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resType)
	}

	var base64Str string
	if err := c.rpc.Call(method, args, &base64Str); err != nil {
		return nil, err
	}

	content, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 attachment: %w", err)
	}
	return content, nil
}
