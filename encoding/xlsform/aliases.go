package xlsform

import (
	"fmt"
	"strings"
)

// pyxform 4.5.0 reads sheet and column names case-insensitively, and accepts older aliases for
// some columns and select types. The decoder maps them to the names the rest of cueform uses, so
// forms that pyxform accepts can be imported.

var (
	// known columns and pyxform's aliases (aliases.survey_header), by snake_case name
	surveyColumnAliases = map[string]string{
		"type": "type", "command": "type",
		"name": "name", "tag": "name", "value": "name",
		"label": "label", "caption": "label",
		"hint": "hint", "guidance_hint": "guidance_hint",
		"required": "required", "required_message": "required_message", "requiredmsg": "required_message",
		"relevant": "relevant", "relevance": "relevant",
		"constraint": "constraint", "constraint_message": "constraint_message", "constraining_message": "constraint_message",
		"read_only": "read_only", "readonly": "read_only",
		"calculation": "calculation", "calculate": "calculation",
		"repeat_count": "repeat_count", "count": "repeat_count", "jr:count": "repeat_count",
		"appearance": "appearance", "default": "default", "choice_filter": "choice_filter",
		"parameters": "parameters", "trigger": "trigger",
		"image": "image", "big-image": "big-image", "audio": "audio", "video": "video",
	}
	// aliases.list_header
	choiceColumnAliases = map[string]string{
		"list_name": "list_name",
		"name":      "name", "value": "name",
		"label": "label", "caption": "label",
		"image": "image", "big-image": "big-image", "audio": "audio", "video": "video",
	}
	// aliases.settings_header and the settings pyxform knows
	settingColumnAliases = map[string]string{
		"form_title": "form_title", "set_form_title": "form_title",
		"form_id": "form_id", "set_form_id": "form_id",
		"version": "version", "default_language": "default_language", "style": "style",
		"public_key": "public_key", "submission_url": "submission_url", "instance_name": "instance_name",
		"auto_send": "auto_send", "auto_delete": "auto_delete", "allow_choice_duplicates": "allow_choice_duplicates",
		"clean_text_values": "clean_text_values", "omit_instanceid": "omit_instanceID", "client_editable": "client_editable",
		"prefix": "prefix",
	}
	columnAliases = map[string]map[string]string{
		surveySheetName:   surveyColumnAliases,
		choiceSheetName:   choiceColumnAliases,
		settingsSheetName: settingColumnAliases,
	}

	// aliases.select, longest first so "select one from file" wins over "select one"
	selectTypeAliases = []struct{ alias, canonical string }{
		{"add select multiple prompt using", "select_multiple"},
		{"add select one prompt using", "select_one"},
		{"select multiple from file", "select_multiple_from_file"},
		{"select all that apply from", "select_multiple"},
		{"select one from file", "select_one_from_file"},
		{"select all that apply", "select_multiple"},
		{"select one from", "select_one"},
		{"select one", "select_one"},
		{"select1", "select_one"},
	}
)

// toSnakeCase is pyxform's to_snake_case
func toSnakeCase(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), "_"))
}

// canonicalHeaders renames a sheet's known columns and their aliases to cueform's names, keeping
// any ::lang suffix. Unknown columns are kept as written, since choice_filter expressions refer to
// choices columns by name.
func canonicalHeaders(sheet string, headers []string) ([]string, error) {
	aliases := columnAliases[sheet]
	out := make([]string, len(headers))
	seen := map[string]string{}
	for i, header := range headers {
		base, lang, translated := strings.Cut(header, "::")
		name := header
		if canonical, ok := aliases[toSnakeCase(base)]; ok {
			name = canonical
			if translated {
				name += "::" + strings.TrimSpace(lang)
			}
		}
		if other, ok := seen[name]; ok && name != "" {
			return nil, fmt.Errorf("%w: on the %s sheet, %q and %q are the same column", ErrInvalidXLSFormSheet, sheet, other, header)
		}
		seen[name] = header
		out[i] = name
	}
	return out, nil
}

// canonicalSelectType rewrites pyxform's older select spellings, such as "select one fruit", to
// "select_one fruit"
func canonicalSelectType(value string) string {
	lower := strings.ToLower(value)
	for _, a := range selectTypeAliases {
		if strings.HasPrefix(lower, a.alias+" ") {
			return a.canonical + value[len(a.alias):]
		}
	}
	return value
}
