package usage

import (
	"os"
	"path/filepath"
)

// writeTemp writes content to a temp file and returns its path.
func writeTemp(content string) (string, error) {
	dir, err := os.MkdirTemp("", "usage-test-*")
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "usage.yaml")
	return p, os.WriteFile(p, []byte(content), 0o600)
}
