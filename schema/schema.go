// Package schema embeds the CUE schema definitions for cueform forms.
package schema

import _ "embed"

// XLSForm is the source of the xlsform schema package (xlsform/schema.cue).
//
//go:embed xlsform/schema.cue
var XLSForm []byte
