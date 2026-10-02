package cmd

import (
	"os"
	"path/filepath"
)

func writeFile(parentDir, file string, b []byte) (string, error) {
	out := filepath.Join(parentDir, file)
	err := os.WriteFile(out, b, 0o644)
	return out, err
}
