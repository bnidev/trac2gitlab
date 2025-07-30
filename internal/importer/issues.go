package importer

import (
	"trac2gitlab/pkg/gitlab"
	"encoding/json"
	"log/slog"
)

func ImportIssues(client *gitlab.Client, projectID any) error {

	project, err := client.GetProject(projectID)
	if err != nil {
		return err
	}

	slog.Info("Starting issue import...", "project", project.Name, "projectID", project.ID)

	return nil
}

type IssueRaw struct {
	ID          int    `json:"ID"`
	TimeCreated string `json:"TimeCreated"`
	TimeChanged string `json:"TimeChanged"`
	Attributes  struct {
		Summary     string `json:"summary"`
		Time        string `json:"time"`
		Owner       string `json:"owner"`
		Reporter    string `json:"reporter"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Milestone  string `json:"milestone"`
	} `json:"Attributes"`
}

type IssueFlat struct {
	ID          int
	Title       string
	Description string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	Owner       string
	Status      string
	MileStoneID  int
}

func ConvertToFlatIssue(data []byte, client *gitlab.Client, projectID any) (*IssueFlat, error) {
	var raw IssueRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal raw issue: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, raw.TimeCreated)
	if err != nil {
		return nil, fmt.Errorf("parse CreatedAt: %w", err)
	}

	updatedAt, err := time.Parse(time.RFC3339, raw.TimeChanged)
	if err != nil {
		return nil, fmt.Errorf("parse UpdatedAt: %w", err)
	}

	milestone, err := client.GetMilestoneByName(projectID, raw.Attributes.Milestone)

	var milestoneID int
	if err != nil || milestone == nil {
		milestoneID = 0
	} else {
		milestoneID = milestone.ID
	}

	flat := &IssueFlat{
		ID:          raw.ID,
		Title:       raw.Attributes.Summary,
		CreatedAt:   &createdAt,
		UpdatedAt:   &updatedAt,
		Owner:       raw.Attributes.Owner,
		Description: raw.Attributes.Description,
		Status:      raw.Attributes.Status,
		MileStoneID: milestoneID,
	}

	return flat, nil
}
