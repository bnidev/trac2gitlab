package gitlab

import (
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// ISOTime is a type alias for time.Time that represents a time in ISO 8601 format.
type ISOTime time.Time

// ListMilestonesOptions defines the options for listing milestones in a project. Here it is aliased to the GitLab client type for easier usage.
type ListMilestonesOptions = gitlab.ListMilestonesOptions

// ListMilestones retrieves a list of milestones for a given project.
func (c *Client) ListMilestones(projectID any, opts *ListMilestonesOptions) ([]*Milestone, error) {
	milestones, _, err := c.git.Milestones.ListMilestones(projectID, opts)
	if err != nil {
		return nil, err
	}

	return milestones, nil
}

// Milestone represents a GitLab milestone, here it is aliased to the GitLab client type for easier usage.
type Milestone = gitlab.Milestone

// GetMilestone retrieves a specific milestone by its ID.
func (c *Client) GetMilestone(projectID any, milestoneID int) (*Milestone, error) {
	milestone, _, err := c.git.Milestones.GetMilestone(projectID, milestoneID)
	if err != nil {
		return nil, err
	}

	return milestone, nil
}

// MilestoneOptions defines the options for creating a new milestone in a project.
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

// UpdateMilestoneOptions defines the options for updating a milestone (attributes inferred from the original GitLab client-go package)
type UpdateMilestoneOptions = gitlab.UpdateMilestoneOptions

// UpdateMilestone updates an existing milestone in the specified project.
func (c *Client) UpdateMilestone(projectID any, milestoneID int, opts *UpdateMilestoneOptions) (*Milestone, error) {
	milestone, _, err := c.git.Milestones.UpdateMilestone(projectID, milestoneID, opts)
	if err != nil {
		return nil, err
	}
	return milestone, nil
}

// GetMilestoneByName retrieves a milestone by its name from the specified project.
func (c *Client) GetMilestoneByName(projectID any, name string) (*Milestone, error) {
	opts := &ListMilestonesOptions{
		Search: &name,
	}

	milestones, _, err := c.git.Milestones.ListMilestones(projectID, opts)
	if err != nil {
		return nil, err
	}

	if len(milestones) == 0 {
		return nil, nil
	}

	for _, milestone := range milestones {
		if milestone.Title == name {
			return milestone, nil
		}
	}

	return nil, nil
}
