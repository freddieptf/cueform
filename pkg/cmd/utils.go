package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
)

func writeFile(parentDir, file string, b []byte) (string, error) {
	out := filepath.Join(parentDir, file)
	err := os.WriteFile(out, b, fs.ModePerm)
	return out, err
}
