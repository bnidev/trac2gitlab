package utils_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fmt"

	"github.com/bnidev/trac2gitlab/internal/utils"
)

func TestReadFilesFromDir(t *testing.T) {
	dir := t.TempDir()

	// Helper to create a file nith content
	createFile := func(name, content string) {
		err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	// Create some files and a subdir
	createFile("file1.json", `{"foo":1}`)
	createFile("file2.md", "# title")
	createFile("file3.txt", "ignored")
	fileErr := os.Mkdir(filepath.Join(dir, "subdir"), 0755)
	if fileErr != nil {
		t.Fatalf("failed to create subdir: %v", fileErr)
	}

	// Also create a file with no read permission (to test skipping)
	noReadFile := filepath.Join(dir, "file4.json")
	err := os.WriteFile(noReadFile, []byte("unreadable"), 0000)
	if err != nil {
		t.Fatalf("failed to create unreadable file: %v", err)
	}
	defer func() {
		if err = os.Chmod(noReadFile, 0644); err != nil {
			t.Fatalf("failed to restore permissions for %s: %v", noReadFile, err)
		}
	}()

	t.Run("Default filetype is json", func(t *testing.T) {
		files, err := utils.ReadFilesFromDir(dir, "", io.Discard)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("expected 1 json file, got %d", len(files))
		}
		if !strings.Contains(string(files[0]), `"foo":1`) {
			t.Errorf("unexpected file content: %s", string(files[0]))
		}
	})

	t.Run("Explicit md filetype", func(t *testing.T) {
		files, err := utils.ReadFilesFromDir(dir, ".md", io.Discard)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("expected 1 md file, got %d", len(files))
		}
		if !strings.Contains(string(files[0]), "title") {
			t.Errorf("unexpected file content: %s", string(files[0]))
		}
	})

	t.Run("Filetype without dot", func(t *testing.T) {
		files, err := utils.ReadFilesFromDir(dir, "md", io.Discard)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("expected 1 md file, got %d", len(files))
		}
	})

	t.Run("Unsupported filetype", func(t *testing.T) {
		_, err := utils.ReadFilesFromDir(dir, ".txt", io.Discard)
		if err == nil {
			t.Fatal("expected error for unsupported filetype")
		}
	})

	t.Run("Skip directories", func(t *testing.T) {
		files, err := utils.ReadFilesFromDir(dir, ".json", io.Discard)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, content := range files {
			if strings.Contains(string(content), "subdir") {
				t.Errorf("directory content included unexpectedly")
			}
		}
	})

	t.Run("Skip unreadable files", func(t *testing.T) {
		files, err := utils.ReadFilesFromDir(dir, ".json", io.Discard)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The unreadable file should be skipped, so total json files read should be 1 (file1.json)
		if len(files) != 1 {
			t.Errorf("expected 1 json file, got %d", len(files))
		}
	})

	t.Run("File limit enforced", func(t *testing.T) {
		// Create more than 1000 json files
		for i := range 1100 {
			fname := filepath.Join(dir, "file"+fmt.Sprint(i)+"_limit.json")
			if err = os.WriteFile(fname, []byte(`{"num":`+fmt.Sprint(i)+`}`), 0644); err != nil {
				t.Fatalf("failed to create test file %s: %v", fname, err)
			}
		}
		files, err := utils.ReadFilesFromDir(dir, ".json", io.Discard)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) > 1000 {
			t.Errorf("expected at most 1000 files, got %d", len(files))
		}
	})
}
