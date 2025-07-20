package trac

import (
	"fmt"
	"slices"
	"time"
	"trac2gitlab/internal/utils"
)

// Ticket represents a single Trac ticket
type Ticket struct {
	ID          int64
	TimeCreated time.Time
	TimeChanged time.Time
	Attributes  map[string]any
	Attachments []Attachment
	History     []ChangeLogEntry
}

// Attachment represents a file attached to a Trac ticket
type Attachment struct {
	Filename    string
	Description string
	Size        int64
	Time        time.Time
	Author      string
}

// GetAllTicketIDs queries Trac for all matching ticket IDs
func (c *Client) GetAllTicketIDs(query string) ([]int, error) {
	var result []int
	err := c.rpc.Call("ticket.query", []any{query}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to query tickets: %w", err)
	}
	return result, nil
}

// GetTicket fetches full ticket data by ID
func (c *Client) GetTicket(id int) (*Ticket, error) {
	var resp []any
	err := c.rpc.Call("ticket.get", []any{id}, &resp)
	if err != nil {
		return nil, fmt.Errorf("ticket.get call failed: %w", err)
	}

	ticketID, ok := resp[0].(int64)
	if !ok {
		return nil, fmt.Errorf("unexpected ticket ID type: %T", resp[0])
	}

	timeCreated, err := utils.ParseRequiredTracTime(resp[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse required created time: %w", err)
	}

	timeChanged, err := utils.ParseRequiredTracTime(resp[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse required changed time: %w", err)
	}

	attributes, ok := resp[3].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected attributes type: %T", resp[3])
	}

	if desc, ok := attributes["description"].(string); ok {
		attributes["description"] = utils.TracToMarkdown(desc)
	}

	attachments, err := c.GetAttachmentList(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments for ticket %d: %w", id, err)
	}

	history, err := c.GetTicketHistory(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket history for %d: %w", id, err)
	}

	return &Ticket{
		ID:          ticketID,
		TimeCreated: timeCreated,
		TimeChanged: timeChanged,
		Attributes:  attributes,
		Attachments: attachments,
		History:     history,
	}, nil
}

// GetAttachmentList retrieves all attachments for a given ticket ID
func (c *Client) GetAttachmentList(id int) ([]Attachment, error) {
	var resp []any
	error := c.rpc.Call("ticket.listAttachments", []any{id}, &resp)
	if error != nil {
		return nil, fmt.Errorf("ticket.listAttachments call failed: %w", error)
	}
	if id <= 0 {
		return nil, fmt.Errorf("invalid ticket ID: %d", id)
	}
	attachments := make([]Attachment, len(resp))
	for i, item := range resp {
		// Each item is expected to be a slice according to Trac Docs: [filename, description, size, time, author]
		fields, ok := item.([]any)
		if !ok || len(fields) != 5 {
			return nil, fmt.Errorf("unexpected attachment format at index %d", i)
		}

		time, _ := utils.ParseRequiredTracTime(fields[3])
		desc := ""
		if fields[1] != nil {
			desc = fmt.Sprintf("%v", fields[1])
		}

		attachments[i] = Attachment{
			Filename:    fmt.Sprintf("%v", fields[0]),
			Description: desc,
			Size:        fields[2].(int64),
			Time:        time,
			Author:      fmt.Sprintf("%v", fields[4]),
		}
	}
	return attachments, nil
}

// GetAttachment retrieves a specific attachment by ticket ID and filename
func (c *Client) GetAttachment(id int, filename string) ([]byte, error) {
	var resp []byte
	err := c.rpc.Call("ticket.getAttachment", []any{id, filename}, &resp)
	if err != nil {
		return nil, fmt.Errorf("ticket.getAttachment call failed: %w", err)
	}
	if id <= 0 || filename == "" {
		return nil, fmt.Errorf("invalid ticket ID or filename: %d, %s", id, filename)
	}
	return resp, nil
}

// ChangeLogEntry represents a single entry in the ticket change log
type ChangeLogEntry struct {
	Time      time.Time
	Author    string
	Field     string
	OldValue  string
	NewValue  string
	Permanent int64
}

// GetTicketHistory retrieves the change log for a specific ticket ID
func (c *Client) GetTicketHistory(id int) ([]ChangeLogEntry, error) {
	var resp []any
	err := c.rpc.Call("ticket.changeLog", []any{id}, &resp)
	if err != nil {
		return nil, fmt.Errorf("ticket.changeLog call failed: %w", err)
	}
	entries := make([]ChangeLogEntry, len(resp))
	for i, item := range resp {
		fields, ok := item.([]any)
		if !ok || len(fields) != 6 {
			return nil, fmt.Errorf("unexpected changelog format at index %d", i)
		}

		time, _ := utils.ParseRequiredTracTime(fields[0])

		oldValue := utils.TracToMarkdown(fmt.Sprintf("%v", fields[3]))
		newValue := utils.TracToMarkdown(fmt.Sprintf("%v", fields[4]))

		entries[i] = ChangeLogEntry{
			Time:      time,
			Author:    fmt.Sprintf("%v", fields[1]),
			Field:     fmt.Sprintf("%v", fields[2]),
			OldValue:  oldValue,
			NewValue:  newValue,
			Permanent: fields[5].(int64),
		}
	}

	for i := 0; i < len(entries); {
		if entries[i].Field != "description" {
			entries = slices.Delete(entries, i, i+1)
		} else {
			i++
		}
	}

	return entries, nil
}
