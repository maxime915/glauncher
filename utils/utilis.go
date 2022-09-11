package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolvePath replace ~/ prefix to produce an absolute path
func ResolvePath(path string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	// make relative without alias
	path = strings.TrimPrefix(path, "~/")

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// make absolute
	return filepath.Join(home, path), nil
}
