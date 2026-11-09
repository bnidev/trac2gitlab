// Package preflight performs read-only migration readiness checks.
package preflight

import (
	"fmt"
	"strings"
	"time"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/identity"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

type Report struct {
	Verdict      string
	TracOK       bool
	GitLabOK     bool
	TimestampOK  bool
	Blocked      bool
	Role         string
	TargetIssues int
	Unresolved   int
	Errors       []string
	Warnings     []string
}

func (r *Report) fail(format string, args ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

func (r *Report) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

func (r *Report) evaluate() {
	switch {
	case r.Blocked:
		r.Verdict = "BLOCKED"
	case len(r.Errors) > 0:
		r.Verdict = "NOT READY"
	case len(r.Warnings) > 0:
		r.Verdict = "READY WITH WARNINGS"
	default:
		r.Verdict = "READY"
	}
}

func Run(cfg *config.Config) *Report {
	report := &Report{}

	tracClient, err := trac.NewTracClient(cfg)
	if err != nil {
		report.fail("create Trac client: %v", err)
	} else {
		if err := tracClient.ValidateExpectedMethods(); err != nil {
			report.fail("Trac XML-RPC: %v", err)
		} else {
			report.TracOK = true
		}
		if err := tracClient.ValidatePluginVersion(); err != nil {
			report.warn("Trac XML-RPC plugin: %v", err)
		}
	}

	client, err := gitlab.NewGitLabClient(cfg)
	if err != nil {
		report.fail("create GitLab client: %v", err)
	} else {
		projectRef := any(cfg.GitLab.ProjectID)
		if cfg.GitLab.ProjectID == 0 {
			projectRef = cfg.GitLab.Project
		}
		project, err := client.GetProject(projectRef)
		if err != nil {
			report.fail("GitLab project: %v", err)
		} else {
			report.GitLabOK = true
			checkGitLabRole(report, client, project.ID)
			issues, err := client.ListProjectIssues(project.ID)
			if err != nil {
				report.fail("list target issues: %v", err)
			} else {
				report.TargetIssues = len(issues)
				if report.TargetIssues > 0 {
					report.warn("target project already has issues; exact Trac-to-GitLab IID parity requires a fresh empty project")
				}
			}
		}
	}

	users, err := identity.LoadFile(cfg.ImportOptions.UsersFile)
	if err != nil {
		report.fail("identity mapping: %v", err)
	} else {
		report.Unresolved = len(users.Unresolved())
		if report.Unresolved > 0 {
			report.warn("%d unresolved identities; their content gets an explicit footer", report.Unresolved)
		}
	}

	report.evaluate()
	return report
}

func checkGitLabRole(report *Report, client *gitlab.Client, projectID any) {
	current, err := client.GetCurrentUser()
	if err != nil {
		report.fail("get current GitLab user: %v", err)
		return
	}
	if current.IsAdmin {
		report.Role = "Administrator"
		report.TimestampOK = true
		return
	}

	members, err := client.GetProjectMembers(projectID)
	if err != nil {
		report.fail("list project members: %v", err)
		return
	}
	report.Role = "None"
	for _, member := range members {
		if member.ID == current.ID {
			level := int(member.AccessLevel)
			report.Role = roleName(level)
			report.TimestampOK = level >= 50
			break
		}
	}
	if !report.TimestampOK {
		report.Blocked = true
		report.fail("current role %s cannot set historical timestamps; Owner or Administrator is required", report.Role)
	}
}

func roleName(level int) string {
	switch level {
	case 50:
		return "Owner"
	case 40:
		return "Maintainer"
	case 30:
		return "Developer"
	case 20:
		return "Reporter"
	case 10:
		return "Guest"
	case 5:
		return "Minimal access"
	default:
		return "None"
	}
}

func (r *Report) Markdown(project string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Preflight — %s\n\n", project)
	fmt.Fprintf(&b, "**Date:** %s\n\n", time.Now().UTC().Format(time.RFC3339))
	b.WriteString("| Check | Result |\n")
	b.WriteString("|---|---|\n")
	fmt.Fprintf(&b, "| Trac XML-RPC | %s |\n", check(r.TracOK))
	fmt.Fprintf(&b, "| GitLab project | %s |\n", check(r.GitLabOK))
	fmt.Fprintf(&b, "| Current role | %s |\n", r.Role)
	fmt.Fprintf(&b, "| Historical timestamps | %s |\n", check(r.TimestampOK))
	fmt.Fprintf(&b, "| Target issues | %d |\n", r.TargetIssues)
	fmt.Fprintf(&b, "| Unresolved identities | %d |\n", r.Unresolved)
	b.WriteString("| Author patch | manual (`trac2gitlab patch`) |\n\n")

	if len(r.Errors) > 0 {
		b.WriteString("## Errors\n\n")
		for _, msg := range r.Errors {
			fmt.Fprintf(&b, "- ✗ %s\n", msg)
		}
		b.WriteString("\n")
	}
	if len(r.Warnings) > 0 {
		b.WriteString("## Warnings\n\n")
		for _, msg := range r.Warnings {
			fmt.Fprintf(&b, "- ⚠ %s\n", msg)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "STATUS: %s\n", r.Verdict)
	return b.String()
}

func check(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
