package server

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Project summarizes one migration workspace on disk.
type Project struct {
	Name          string `json:"name"`
	Dir           string `json:"-"`
	HasConfig     bool   `json:"hasConfig"`
	HasState      bool   `json:"hasState"`
	HasExport     bool   `json:"hasExport"`
	HasPreflight  bool   `json:"hasPreflight"`
	HasValidation bool   `json:"hasValidation"`
}

// ListProjects scans root for directories containing migration.yaml or
// config.yaml. Missing root yields an empty list. "." itself is included when
// it holds a config file.
func ListProjects(root string) []Project {
	seen := map[string]bool{}
	var out []Project

	add := func(dir string) {
		abs, err := filepath.Abs(dir)
		if err != nil {
			abs = dir
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		name := filepath.Base(abs)
		if dir == "." || dir == root {
			if abs != "" {
				name = filepath.Base(abs)
			}
		}
		out = append(out, Project{
			Name:          name,
			Dir:           abs,
			HasConfig:     hasFile(dir, "migration.yaml") || hasFile(dir, "config.yaml"),
			HasState:      hasFile(dir, "state", "migration.yaml"),
			HasExport:     hasExport(dir),
			HasPreflight:  hasFile(dir, "reports", "preflight.md"),
			HasValidation: hasFile(dir, "reports", "validation.md"),
		})
	}

	if hasFile(root, "migration.yaml") || hasFile(root, "config.yaml") {
		add(root)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if hasFile(dir, "migration.yaml") || hasFile(dir, "config.yaml") {
			add(dir)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ResolveProject maps a ?project= name to a directory under root. Empty name
// selects root itself when it holds a config, otherwise the first project.
func ResolveProject(root, name string) string {
	projects := ListProjects(root)
	if len(projects) == 0 {
		return root
	}
	if name == "" {
		return projects[0].Dir
	}
	// Reject anything that is not a plain project name (no slashes or dots),
	// so ?project=../../x cannot escape root. LoadProject then fails on the
	// returned root, which the handlers already turn into a 400.
	if !projectNameRe.MatchString(name) {
		return root
	}
	for _, p := range projects {
		if p.Name == name {
			return p.Dir
		}
	}
	return filepath.Join(root, name)
}

func hasFile(dir string, parts ...string) bool {
	_, err := os.Stat(filepath.Join(append([]string{dir}, parts...)...))
	return err == nil
}

// hasExport reports whether the project's export directory (export_options.
// export_dir, default data/) exists and holds anything. It reads the config
// file directly: config.LoadProject would load .env into the server process.
func hasExport(dir string) bool {
	var cfg struct {
		ExportOptions struct {
			ExportDir string `yaml:"export_dir"`
		} `yaml:"export_options"`
	}
	for _, name := range []string{"migration.yaml", "config.yaml"} {
		if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			_ = yaml.Unmarshal(b, &cfg)
			break
		}
	}
	exportDir := cfg.ExportOptions.ExportDir
	if exportDir == "" {
		exportDir = "data"
	}
	if !filepath.IsAbs(exportDir) {
		exportDir = filepath.Join(dir, exportDir)
	}
	entries, err := os.ReadDir(exportDir)
	return err == nil && len(entries) > 0
}
