package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	"trac2gitlab/internal/config"
	"trac2gitlab/internal/utils"
	"trac2gitlab/pkg/gitlab"
)

func ImportIssues(client *gitlab.Client, config *config.Config) error {

	project, err := client.GetProject(config.GitLab.ProjectID)
	if err != nil {
		return err
	}

	slog.Info("Starting issue import...", "project", project.Name, "projectID", project.ID)

	existingIssues, err := client.ListProjectIssues(project.ID)
	if err != nil {
		return fmt.Errorf("failed to list existing issues: %w", err)
	}

	if len(existingIssues) > 0 {
		slog.Debug("Existing issues found", "count", len(existingIssues))
	} else {
		slog.Debug("No existing issues found for project", "projectID", project.ID)
	}

	fmt.Printf("Importing issues for project: %s (ID: %d)\n", project.Name, project.ID)

	issues, err := utils.ReadFilesFromDir("data/tickets", ".json")
	if err != nil {
		return fmt.Errorf("failed to read milestones from directory: %w", err)
	}

	for _, issueData := range issues {
		flat, err := ConvertToFlatIssue(issueData, client, project.ID)
		if err != nil {
			return fmt.Errorf("failed to process issue: %w", err)
		}

		if existingIssue, err := client.GetIssue(project.ID, flat.ID); err != nil {
			slog.Debug("Importing new issue", "ID", flat.ID, "Title", flat.Title)

			// Create the issue in GitLab
			_, err := client.CreateIssue(project.ID, &gitlab.CreateIssueOptions{
				IID:         &flat.ID,
				Title:       &flat.Title,
				Description: &flat.Description,
				CreatedAt:   flat.CreatedAt,
				MilestoneID: &flat.MileStoneID,
			})
			if err != nil {
				return fmt.Errorf("failed to create issue %d: %w", flat.ID, err)
			}

			// Update the issue status, because it cant be set on create
			if flat.Status == "closed" {
				slog.Debug("Setting issue status to closed", "ID", flat.ID, "Title", flat.Title)
				var stateEvent = "close"
				_, err = client.UpdateIssue(project.ID, flat.ID, &gitlab.UpdateIssueOptions{
					StateEvent: &stateEvent,
				})
				if err != nil {
					return fmt.Errorf("failed to close issue %d: %w", flat.ID, err)
				}
			}

			slog.Debug("Issue created successfully", "ID", flat.ID, "Title", flat.Title)

		} else {
			slog.Debug("Issue already exists, checking for updates", "ID", flat.ID, "Title", flat.Title)

			updateOpts := &gitlab.UpdateIssueOptions{}

			needsUpdate := false
			if existingIssue.Title != flat.Title {
				updateOpts.Title = &flat.Title
				needsUpdate = true
			}

			if existingIssue.Description != flat.Description {
				updateOpts.Description = &flat.Description
				needsUpdate = true
			}

			// if existingIssue.UpdatedAt != flat.UpdatedAt {
			// 	updateOpts.UpdatedAt = flat.UpdatedAt
			// 	needsUpdate = true
			// }

			// INFO: Re-Opening/Closing issues will show up as updates in other GitLab issues (thus their updated_at will change)
			if existingIssue.State != "closed" && flat.Status == "closed" {
				var stateEvent = "close"
				updateOpts.StateEvent = &stateEvent
				needsUpdate = true
			}

			if existingIssue.State == "closed" && flat.Status != "closed" {
				var stateEvent = "reopen"
				updateOpts.StateEvent = &stateEvent
				needsUpdate = true
			}

			if existingIssue.Milestone == nil || existingIssue.Milestone.ID != flat.MileStoneID {
				if flat.MileStoneID != 0 {
					updateOpts.MilestoneID = &flat.MileStoneID
					needsUpdate = true
				}
			}

			if needsUpdate {
				slog.Debug("Updating existing issue", "ID", flat.ID, "Title", flat.Title)

				updateOpts.UpdatedAt = flat.UpdatedAt
				_, err := client.UpdateIssue(config.GitLab.ProjectID, flat.ID, updateOpts)
				if err != nil {
					fmt.Print(updateOpts)
					return fmt.Errorf("failed to update issue %d: %w", flat.ID, err)
				}
			} else {
				slog.Debug("No updates needed for existing issue", "ID", flat.ID, "Title", flat.Title)
			}

		}

	}
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
		Milestone   string `json:"milestone"`
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
	MileStoneID int
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
