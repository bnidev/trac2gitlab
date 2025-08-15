package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadFilesFromDir reads all files with the given extension from the specified directory.
func ReadFilesFromDir(dirPath string, fileType string, warnWriter io.Writer) ([][]byte, error) {
	allowedTypes := map[string]bool{".json": true, ".md": true}
	if fileType == "" {
		fileType = ".json"
	}
	if !strings.HasPrefix(fileType, ".") {
		fileType = "." + fileType
	}
	if !allowedTypes[fileType] {
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var results [][]byte
	const maxFiles = 1000
	for i, entry := range entries {
		if i >= maxFiles {
			break
		}
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), fileType) {
			fullPath := filepath.Join(dirPath, entry.Name())
			content, err := os.ReadFile(fullPath)
			if err != nil {
				if warnWriter != nil {
					if _, err = fmt.Fprintf(warnWriter, "Warning: failed to read file %s: %v\n", fullPath, err); err != nil {
						return nil, fmt.Errorf("failed to write warning: %w", err)
					}
				}
				continue
			}
			results = append(results, content)
		}
	}

	return results, nil
}
