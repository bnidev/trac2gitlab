package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/references"
	"github.com/bnidev/trac2gitlab/internal/utils"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
)

// Result summarises one import phase.
type Result struct {
	Total     int
	Completed int
	Failed    int
	Errors    []string
}

func (r *Result) fail(err error) {
	r.Failed++
	r.Errors = append(r.Errors, err.Error())
}

func ImportIssues(client *gitlab.Client, config *config.Config) (Result, error) {
	var result Result

	refs, err := references.Load(references.Options{
		Enabled:        config.References.Enabled,
		TicketsFile:    config.References.TicketsFile,
		ChangesetsFile: config.References.ChangesetsFile,
		BaseURL:        config.GitLab.BaseURL,
		ProjectPath:    config.GitLab.ProjectPath,
	})
	if err != nil {
		return result, err
	}

	project, err := client.GetProject(config.GitLab.ProjectID)
	if err != nil {
		return result, err
	}

	slog.Info("Starting issue import...", "project", project.Name, "projectID", project.ID)

	existingIssues, err := client.ListProjectIssues(project.ID)
	if err != nil {
		return result, fmt.Errorf("failed to list existing issues: %w", err)
	}
	slog.Debug("Existing issues", "count", len(existingIssues))

	fmt.Printf("Importing issues for project: %s (ID: %d)\n", project.Name, project.ID)

	issues, err := utils.ReadFilesFromDir(filepath.Join(config.ExportOptions.ExportDir, "tickets"), ".json", os.Stderr)
	if err != nil {
		return result, fmt.Errorf("read tickets: %w", err)
	}
	result.Total = len(issues)

	for _, issueData := range issues {
		flat, err := ConvertToFlatIssue(issueData, client, project.ID)
		if err != nil {
			result.fail(err)
			return result, fmt.Errorf("failed to process issue: %w", err)
		}
		flat.Title = refs.Rewrite(flat.Title)
		flat.Description = refs.Rewrite(flat.Description)

		existingIssue, err := client.GetIssue(project.ID, flat.ID)
		if err != nil {
			slog.Debug("Importing new issue", "ID", flat.ID, "Title", flat.Title)
			if config.DryRun {
				slog.Info("dry-run: would create issue", "ID", flat.ID, "Title", flat.Title)
				result.Completed++
				continue
			}

			// Create as the admin client: only admin/owner can set IID and CreatedAt.
			// Author attribution is fixed by the patch pass.
			created, err := client.CreateIssue(project.ID, &gitlab.CreateIssueOptions{
				IID:         &flat.ID,
				Title:       &flat.Title,
				Description: &flat.Description,
				CreatedAt:   flat.CreatedAt,
				MilestoneID: &flat.MileStoneID,
			})
			if err != nil {
				result.fail(err)
				return result, fmt.Errorf("failed to create issue %d: %w", flat.ID, err)
			}
			if created.IID != flat.ID {
				err := fmt.Errorf("issue %d was created with IID %d: admin/owner rights are required to preserve IIDs", flat.ID, created.IID)
				result.fail(err)
				return result, err
			}

			// Update the issue status, because it cant be set on create.
			// UpdatedAt backdates closed_at (and the close system note/state event).
			if flat.Status == "closed" {
				slog.Debug("Setting issue status to closed", "ID", flat.ID, "Title", flat.Title)
				stateEvent := "close"
				_, err = client.UpdateIssue(project.ID, flat.ID, &gitlab.UpdateIssueOptions{
					StateEvent: &stateEvent,
					UpdatedAt:  flat.UpdatedAt,
				})
				if err != nil {
					result.fail(err)
					return result, fmt.Errorf("failed to close issue %d: %w", flat.ID, err)
				}
			}

			slog.Debug("Issue created successfully", "ID", flat.ID, "Title", flat.Title)
			result.Completed++
			continue
		}

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
		// INFO: Re-Opening/Closing issues will show up as updates in other GitLab issues (thus their updated_at will change)
		if existingIssue.State != "closed" && flat.Status == "closed" {
			stateEvent := "close"
			updateOpts.StateEvent = &stateEvent
			needsUpdate = true
		}
		if existingIssue.State == "closed" && flat.Status != "closed" {
			stateEvent := "reopen"
			updateOpts.StateEvent = &stateEvent
			needsUpdate = true
		}
		if existingIssue.Milestone == nil || existingIssue.Milestone.ID != flat.MileStoneID {
			if flat.MileStoneID != 0 {
				updateOpts.MilestoneID = &flat.MileStoneID
				needsUpdate = true
			}
		}

		if needsUpdate && config.DryRun {
			slog.Info("dry-run: would update issue", "ID", flat.ID, "Title", flat.Title)
			result.Completed++
			continue
		}
		if needsUpdate {
			slog.Debug("Updating existing issue", "ID", flat.ID, "Title", flat.Title)
			updateOpts.UpdatedAt = flat.UpdatedAt
			if _, err := client.UpdateIssue(config.GitLab.ProjectID, flat.ID, updateOpts); err != nil {
				result.fail(err)
				return result, fmt.Errorf("failed to update issue %d: %w", flat.ID, err)
			}
		} else {
			slog.Debug("No updates needed for existing issue", "ID", flat.ID, "Title", flat.Title)
		}
		result.Completed++
	}

	return result, nil
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
	ID          int64
	Title       string
	Description string
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
	Reporter    string
	Owner       string
	Status      string
	MileStoneID int64
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

	var milestoneID int64
	if err != nil || milestone == nil {
		milestoneID = 0
	} else {
		milestoneID = milestone.ID
	}

	flat := &IssueFlat{
		ID:          int64(raw.ID),
		Title:       raw.Attributes.Summary,
		CreatedAt:   &createdAt,
		UpdatedAt:   &updatedAt,
		Reporter:    raw.Attributes.Reporter,
		Owner:       raw.Attributes.Owner,
		Description: raw.Attributes.Description,
		Status:      raw.Attributes.Status,
		MileStoneID: milestoneID,
	}

	return flat, nil
}
