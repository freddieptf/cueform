package cmd

import (
	"testing"

	"github.com/freddieptf/cueform/encoding/xlsform"
)

// decoded forms import the published schema package unless -pkg says otherwise
func TestDecodeDefaultPackage(t *testing.T) {
	cmd := newDecoderCmd()
	if err := cmd.flag.Parse([]string{"form.xlsx"}); err != nil {
		t.Fatal(err)
	}
	if *cmd.pkg != xlsform.SchemaPackage {
		t.Errorf("have %q, want %q", *cmd.pkg, xlsform.SchemaPackage)
	}
	cmd = newDecoderCmd()
	if err := cmd.flag.Parse([]string{"-pkg", "example.com/schema", "form.xlsx"}); err != nil {
		t.Fatal(err)
	}
	if *cmd.pkg != "example.com/schema" {
		t.Errorf("have %q, want the -pkg value", *cmd.pkg)
	}
}
