package importer

import (
	"fmt"
	"trac2gitlab/pkg/gitlab"
)

func ImportIssues(client *gitlab.Client, projectID any) error {

	project, err := client.GetProject(projectID)
	if err != nil {
		return err
	}

	fmt.Printf("Importing issues for project: %s (ID: %d)\n", project.Name, project.ID)

	return nil
}
