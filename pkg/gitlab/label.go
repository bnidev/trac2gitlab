package gitlab

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type Label = gitlab.Label

func (c *Client) GetProjectLabels(projectID int) ([]*Label, error) {
	labels, _, err := c.git.Labels.ListLabels(projectID, &gitlab.ListLabelsOptions{})
	if err != nil {
		return nil, err
	}

	return labels, nil
}

func (c *Client) CreateLabel(projectID int, opts *gitlab.CreateLabelOptions) (*Label, error) {
	label, _, err := c.git.Labels.CreateLabel(projectID, opts)
	if err != nil {
		return nil, err
	}

	return label, nil
}

func (c* Client) GetLabelByID(projectID int, labelID int) (*Label, error) {
	label, _, err := c.git.Labels.GetLabel(projectID, labelID)
	if err != nil {
		return nil, err
	}

	return label, nil
}

func (c *Client) GetLabelbyName(projectID int, name string) (*Label, error) {
	labels, _, err := c.git.Labels.ListLabels(projectID, &gitlab.ListLabelsOptions{Search: &name})
	if err != nil {
		return nil, err
	}

	if len(labels) == 0 {
		return nil, nil
	}

	for _, label := range labels {
		if label.Name == name {
			return label, nil
		}
	}

	return nil, nil
}

func (c *Client) UpdateLabel(projectID int, labelID int, opts *gitlab.UpdateLabelOptions) (*Label, error) {
	label, _, err := c.git.Labels.UpdateLabel(projectID, labelID, opts)
	if err != nil {
		return nil, err
	}

	return label, nil
}
