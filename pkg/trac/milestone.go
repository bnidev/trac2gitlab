package trac

import (
	"fmt"
	"time"
	"trac2gitlab/internal/utils"
)

// Milestone represents a Trac milestone with its details.
type Milestone struct {
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	CompletedDate *time.Time `json:"completed_date,omitempty"`
}

// GetMilestoneNames retrieves the names of all milestones in Trac.
func (c *Client) GetMilestoneNames() ([]string, error) {
	var resp []string

	err := c.rpc.Call("ticket.milestone.getAll", nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetMilestoneByName retrieves a milestone by its name.
func (c *Client) GetMilestoneByName(name string) (*Milestone, error) {
	var resp map[string]any
	err := c.rpc.Call("ticket.milestone.get", []any{name}, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to call milestone.get: %w", err)
	}

	m := &Milestone{Name: name}

	if desc, ok := resp["description"].(string); ok {
		m.Description = &desc
	}

	if dueRaw, ok := resp["due"]; ok {
		due, err := utils.ParseTracTime(dueRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse due date: %w", err)
		}
		m.DueDate = due
	}

	if completedRaw, ok := resp["completed"]; ok {
		completed, err := utils.ParseTracTime(completedRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse completed date: %w", err)
		}
		m.CompletedDate = completed
	}

	return m, nil
}

