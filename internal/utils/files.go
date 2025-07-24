package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFilesFromDir reads all files with the given extension from the specified directory.
func ReadFilesFromDir(dirPath string, fileType string) ([][]byte, error) {
	if fileType == "" {
		fileType = ".json"
	}
	if !strings.HasPrefix(fileType, ".") {
		fileType = "." + fileType
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var results [][]byte

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(entry.Name(), fileType) {
			fullPath := filepath.Join(dirPath, entry.Name())
			content, err := os.ReadFile(fullPath)
			if err != nil {
				fmt.Printf("Warning: failed to read file %s: %v\n", fullPath, err)
				continue
			}
			results = append(results, content)
		}
	}

	return results, nil
}

