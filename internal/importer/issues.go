package importer

import (
	"trac2gitlab/pkg/gitlab"
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
