package gitlab

import (
	"time"

	"gitlab.com/gitlab-org/api/client-go"
)

type ISOTime time.Time

type ListMilestonesOptions = gitlab.ListMilestonesOptions

// ListMilestones retrieves a list of milestones for a given project.
func (c *Client) ListMilestones(projectID any, opts *ListMilestonesOptions) ([]*Milestone, error) {
	milestones, _, err := c.git.Milestones.ListMilestones(projectID, opts)
	if err != nil {
		return nil, err
	}

	return milestones, nil
}


type Milestone = gitlab.Milestone

// GetMilestone retrieves a specific milestone by its ID.
func (c *Client) GetMilestone(projectID any, milestoneID int) (*Milestone, error) {
	milestone, _, err := c.git.Milestones.GetMilestone(projectID, milestoneID)
	if err != nil {
		return nil, err
	}

	return milestone, nil
}

type MilestoneOptions struct {
	Title       string
	Description string
	DueDate     *ISOTime
}

// CreateMilestone creates a new milestone in the specified project.
func (c *Client) CreateMilestone(projectID any, opts *MilestoneOptions) (*Milestone, error) {
	gitlabOpts := &gitlab.CreateMilestoneOptions{
		Title:       &opts.Title,
		Description: &opts.Description,
		DueDate:     (*gitlab.ISOTime)(opts.DueDate),
	}
	milestone, _, err := c.git.Milestones.CreateMilestone(projectID, gitlabOpts)
	if err != nil {
		return nil, err
	}
	return milestone, nil
}

type UpdateMilestoneOptions = gitlab.UpdateMilestoneOptions

// UpdateMilestone updates an existing milestone in the specified project.
func (c *Client) UpdateMilestone(projectID any, milestoneID int, opts *UpdateMilestoneOptions) (*Milestone, error) {
	milestone, _, err := c.git.Milestones.UpdateMilestone(projectID, milestoneID, opts)
	if err != nil {
		return nil, err
	}
	return milestone, nil
}

