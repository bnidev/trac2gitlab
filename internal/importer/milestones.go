package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/utils"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"

	gitlabClient "gitlab.com/gitlab-org/api/client-go"
)

func ImportMilestones(client *gitlab.Client, config *config.Config) error {

	project, err := client.GetProject(config.GitLab.ProjectID)
	if err != nil {
		return err
	}

	slog.Info("Starting milestone import...", "project", project.Name, "projectID", project.ID)

	existingMilestones, err := client.ListMilestones(project.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to list existing milestones: %w", err)
	}

	if len(existingMilestones) > 0 {
		slog.Debug("Existing milestones found", "count", len(existingMilestones))
	} else {
		slog.Debug("No existing milestones found for project", "projectID", project.ID)
	}

	fmt.Printf("Importing milestones for project: %s (ID: %d)\n", project.Name, project.ID)

	// Import milestones from exported Trac data
	milestones, err := utils.ReadFilesFromDir("data/milestones", ".json")
	if err != nil {
		return fmt.Errorf("failed to read milestones from directory: %w", err)
	}

	for _, milestoneData := range milestones {
		var input struct {
			Name          string `json:"name"`
			Description   string `json:"description"`
			DueDate       string `json:"due_date"`       // format: "2025-07-24T16:00:00Z"
			CompletedDate string `json:"completed_date"` // format: "2025-07-24T16:00:00Z"
		}

		if err := json.Unmarshal(milestoneData, &input); err != nil {
			return fmt.Errorf("failed to unmarshal milestone data: %w", err)
		}

		var dueDate time.Time
		if input.DueDate != "" {
			dueDate, err = time.Parse(time.RFC3339, input.DueDate)
			if err != nil {
				return fmt.Errorf("failed to parse due date %q: %w", input.DueDate, err)
			}
		}

		// Check if input.Name is in the list of existing milestones
		milestoneExists := false
		if len(existingMilestones) > 0 {
			for _, existing := range existingMilestones {
				if existing.Title == input.Name {
					milestoneExists = true
					slog.Debug("Found existing milestone, skipping creation", "title", existing.Title, "id", existing.ID)

					slog.Debug("Checking if existing milestone needs update", "title", existing.Title, "id", existing.ID)
					updateOpts := &gitlab.UpdateMilestoneOptions{}

					needsUpdate := false

					if input.Description != "" && existing.Description != input.Description {
						updateOpts.Description = &input.Description
						needsUpdate = true
					}

					if input.DueDate != "" {
						parsedDueDate, err := gitlabClient.ParseISOTime(dueDate.Format("2006-01-02"))
						if err != nil {
							return fmt.Errorf("failed to parse due date %q: %w", input.DueDate, err)
						}
						if existing.DueDate == nil || *existing.DueDate != parsedDueDate {
							updateOpts.DueDate = &parsedDueDate
							needsUpdate = true
						}
					}

					if input.CompletedDate != "" && existing.State != "closed" {
						var stateEvent = "close"
						updateOpts.StateEvent = &stateEvent
						needsUpdate = true
					}

					if input.CompletedDate == "" && existing.State == "closed" {
						var stateEvent = "activate"
						updateOpts.StateEvent = &stateEvent
						needsUpdate = true
					}

					if needsUpdate {
						slog.Debug("Updating existing milestone", "title", existing.Title, "id", existing.ID)
						_, err = client.UpdateMilestone(project.ID, existing.ID, updateOpts)
						if err != nil {
							return fmt.Errorf("failed to update milestone: %w", err)
						}
						slog.Info("Milestone updated successfully", "title", existing.Title, "id", existing.ID)
					} else {
						slog.Debug("No updates needed for existing milestone", "title", existing.Title, "id", existing.ID)
					}

					break
				}
			}
		}

		if !milestoneExists {
			opts := &gitlab.MilestoneOptions{
				Title:       input.Name,
				Description: input.Description,
				DueDate:     (*gitlab.ISOTime)(&dueDate),
			}

			milestone, err := client.CreateMilestone(project.ID, opts)
			if err != nil {
				return fmt.Errorf("failed to create milestone: %s", err)
			}

			slog.Info("Milestone created successfully", "title", milestone.Title, "id", milestone.ID)

			// close milestone if CompletedDate is provided
			if input.CompletedDate != "" {
				var stateEvent = "close"

				updateOpts := &gitlab.UpdateMilestoneOptions{
					StateEvent: &stateEvent,
				}
				_, err = client.UpdateMilestone(project.ID, milestone.ID, updateOpts)
				if err != nil {
					slog.Error("Failed to close milestone", "title", milestone.Title, "id", milestone.ID, "error", err)
				} else {
					slog.Info("Milestone closed successfully", "title", milestone.Title, "id", milestone.ID)
				}
			}
		}
	}

	slog.Info("Milestone import completed", "count", len(milestones))
	return nil
}
