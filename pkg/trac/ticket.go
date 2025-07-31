package trac

import (
	"fmt"
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
	Comments    []ChangeLogEntry
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

	attachments, err := ListAttachments(c, ResourceTicket, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments for ticket %d: %w", id, err)
	}

	descriptions, comments, err := c.GetTicketHistory(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket history for %d: %w", id, err)
	}

	return &Ticket{
		ID:          ticketID,
		TimeCreated: timeCreated,
		TimeChanged: timeChanged,
		Attributes:  attributes,
		Attachments: attachments,
		History:     descriptions,
		Comments:    comments,
	}, nil
}

// ChangeLogEntry represents a single entry in the ticket change log
type ChangeLogEntry struct {
	Time      time.Time
	Author    string
	Field     string
	OldValue  *string
	NewValue  *string
	Permanent int64
}

// GetTicketHistory retrieves the change log for a specific ticket ID,
// returning description and comment entries separately.
func (c *Client) GetTicketHistory(id int) ([]ChangeLogEntry, []ChangeLogEntry, error) {
	var resp []any
	err := c.rpc.Call("ticket.changeLog", []any{id}, &resp)
	if err != nil {
		return nil, nil, fmt.Errorf("ticket.changeLog call failed: %w", err)
	}

	var descriptions []ChangeLogEntry
	var comments []ChangeLogEntry

	for i, item := range resp {
		fields, ok := item.([]any)
		if !ok || len(fields) != 6 {
			return nil, nil, fmt.Errorf("unexpected changelog format at index %d", i)
		}

		time, _ := utils.ParseRequiredTracTime(fields[0])

		var oldValue, newValue *string
		if fields[3] != nil {
			val := utils.TracToMarkdown(fmt.Sprintf("%v", fields[3]))
			oldValue = &val
		}
		if fields[4] != nil {
			val := utils.TracToMarkdown(fmt.Sprintf("%v", fields[4]))
			newValue = &val
		}

		entry := ChangeLogEntry{
			Time:      time,
			Author:    fmt.Sprintf("%v", fields[1]),
			Field:     fmt.Sprintf("%v", fields[2]),
			OldValue:  oldValue,
			NewValue:  newValue,
			Permanent: fields[5].(int64),
		}

		switch entry.Field {
		case "description":
			descriptions = append(descriptions, entry)
		case "comment":
			comments = append(comments, entry)
		}
	}

	return descriptions, comments, nil
}
