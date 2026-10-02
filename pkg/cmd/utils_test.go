package cmd

import (
	"os"
	"testing"
)

// files the CLI writes are ordinary files, not executables
func TestWriteFileMode(t *testing.T) {
	out, err := writeFile(t.TempDir(), "form.xlsx", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode&0o111 != 0 {
		t.Errorf("%s is executable: %v", out, mode)
	}
}
