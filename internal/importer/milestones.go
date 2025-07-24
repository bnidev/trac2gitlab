package importer

import (
	"encoding/json"
	"fmt"
	"time"
	"trac2gitlab/internal/utils"
	"trac2gitlab/pkg/gitlab"

	gitlabClient "gitlab.com/gitlab-org/api/client-go"
)

func ImportMilestones(client *gitlab.Client, projectID any) error {

	project, err := client.GetProject(projectID)
	if err != nil {
		return err
	}

	existingMilestones, err := client.ListMilestones(projectID, nil)
	if err != nil {
		return fmt.Errorf("failed to list existing milestones: %w", err)
	}

	if len(existingMilestones) > 0 {
		fmt.Printf("Found %d existing milestone%s in project %s (ID: %d)\n", len(existingMilestones), func() string {
			if len(existingMilestones) == 1 {
				return ""
			}
			return "s"
		}(), project.Name, project.ID)
	} else {
		fmt.Printf("No existing milestones found for project %s (ID: %d)\n", project.Name, project.ID)
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
					fmt.Printf("Milestone '%s' already exists (ID: %d), skipping creation.\n", existing.Title, existing.ID)

					// If the milestone already exists, update it if needed
					fmt.Printf("Checking if milestone '%s' needs updating...\n", existing.Title)
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

					if input.CompletedDate != "" && existing.State != "close" {
						var stateEvent = "close"
						updateOpts.StateEvent = &stateEvent
						needsUpdate = true
					}

					if input.CompletedDate == "" && existing.State == "close" {
						var stateEvent = "activate"
						updateOpts.StateEvent = &stateEvent
						needsUpdate = true
					}

					if needsUpdate {
						fmt.Printf("Updating milestone '%s' (ID: %d)\n", existing.Title, existing.ID)
						_, err = client.UpdateMilestone(projectID, existing.ID, updateOpts)
						if err != nil {
							return fmt.Errorf("failed to update milestone: %w", err)
						}
						fmt.Printf("Updated milestone '%s' (ID: %d)\n", existing.Title, existing.ID)
					} else {
						fmt.Printf("No updates needed for milestone '%s' (ID: %d)\n", existing.Title, existing.ID)
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

			milestone, err := client.CreateMilestone(projectID, opts)
			if err != nil {
				fmt.Printf("Failed to create milestone: %v\n", err)
			}

			fmt.Printf("Created milestone: %s (ID: %d)\n", milestone.Title, milestone.ID)

			// close milestone if CompletedDate is provided
			if input.CompletedDate != "" {
				if err != nil {
					return fmt.Errorf("failed to parse completed date %q: %w", input.CompletedDate, err)
				}

				var stateEvent = "close"

				updateOpts := &gitlab.UpdateMilestoneOptions{
					StateEvent: &stateEvent,
				}
				_, err = client.UpdateMilestone(projectID, milestone.ID, updateOpts)
				if err != nil {
					fmt.Printf("Failed to close milestone: %v\n", err)
				} else {
					fmt.Printf("Closed milestone %s\n", milestone.Title)
				}
			}
		}
	}

	return nil
}
