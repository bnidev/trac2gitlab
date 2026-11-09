package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/state"
	"github.com/spf13/cobra"
)

var phaseOrder = []string{"milestones", "issues", "notes", "attachments", "wiki"}

func statusCmd(ctx *app.AppContext) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration progress from state/migration.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := filepath.Join(ctx.ProjectDir, "state", "migration.yaml")
			run, err := state.Load(path)
			if err != nil {
				return err
			}
			if asJSON {
				data, err := json.MarshalIndent(run, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}
			printStatus(ctx.ProjectDir, run)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func printStatus(projectDir string, run *state.Migration) {
	fmt.Printf("Project:  %s\n", projectDir)
	fmt.Printf("Status:   %s\n", run.Status)
	fmt.Printf("Mode:     %s\n", mode(run.DryRun))
	if !run.StartedAt.IsZero() {
		fmt.Printf("Started:  %s\n", run.StartedAt.UTC().Format("2006-01-02 15:04:05Z"))
	}
	if run.FinishedAt != nil {
		fmt.Printf("Finished: %s\n", run.FinishedAt.UTC().Format("2006-01-02 15:04:05Z"))
	}

	names := make([]string, 0, len(run.Phases))
	seen := make(map[string]bool)
	for _, name := range phaseOrder {
		if _, ok := run.Phases[name]; ok {
			names = append(names, name)
			seen[name] = true
		}
	}
	var extra []string
	for name := range run.Phases {
		if !seen[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	names = append(names, extra...)

	if len(names) > 0 {
		fmt.Println("Phases:")
		for _, name := range names {
			phase := run.Phases[name]
			line := fmt.Sprintf("  %-14s %-10s %d/%d", name, phase.Status, phase.Completed, phase.Total)
			if phase.Failed > 0 {
				line += fmt.Sprintf("  failed=%d", phase.Failed)
			}
			fmt.Println(line)
			if len(phase.Errors) > 0 {
				fmt.Printf("    error: %s\n", phase.Errors[0])
				if len(phase.Errors) > 1 {
					fmt.Printf("    ... %d more error(s)\n", len(phase.Errors)-1)
				}
			}
		}
	}

	fmt.Printf("Next:     %s\n", nextStep(projectDir, run))
}

func mode(dryRun bool) string {
	if dryRun {
		return "dry-run"
	}
	return "execute"
}

func nextStep(projectDir string, run *state.Migration) string {
	if run.Status == state.StatusRunning {
		return "migration is running"
	}
	if run.Status == state.StatusFailed {
		return "rerun trac2gitlab migrate (resumes completed objects)"
	}
	if len(run.Phases) == 0 {
		return "run trac2gitlab preflight, export, and migrate --dry-run"
	}
	if run.DryRun {
		return "review the plan, then rerun migrate --execute"
	}
	if fileExists(filepath.Join(projectDir, "reports", "validation.md")) {
		return "review reports/validation.md"
	}
	return "run trac2gitlab patch (if authors matter), then validate"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
