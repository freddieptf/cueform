package xlsform

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// The XLSForm spec recommends naming each language with its code, as in label::English (en), so
// apps can match the form's language to the device's. The encoder warns about every language not
// written that way. Any code is accepted; only the name and the code have to be there.

// a name, one space, and a code in brackets, as in "English (en)" or "Chinese (zh-Hans)"
var recommendedLanguageRe = regexp.MustCompile(`^\S(.*\S)? \([^()\s]+\)$`)

// languagesNotRecommended returns the languages not written as Name (code). "default" is skipped:
// it is pyxform's name for a column with no language next to translated ones, not a name the
// author chose.
func languagesNotRecommended(languages []string) []string {
	var bad []string
	for _, lang := range languages {
		if lang != "default" && !recommendedLanguageRe.MatchString(lang) {
			bad = append(bad, lang)
		}
	}
	return bad
}

// formLanguages returns the languages used by translatable columns on the survey and choices
// sheets, sorted
func formLanguages(headerRows ...[]string) []string {
	var languages []string
	for _, headers := range headerRows {
		for _, header := range headers {
			col, lang, ok := strings.Cut(header, "::")
			if ok && IsTranslatableColumn(col) && !slices.Contains(languages, lang) {
				languages = append(languages, lang)
			}
		}
	}
	slices.Sort(languages)
	return languages
}

// languageWarning returns a warning naming the languages not written as recommended, or ""
func languageWarning(languages []string) string {
	bad := languagesNotRecommended(languages)
	if len(bad) == 0 {
		return ""
	}
	return fmt.Sprintf("warning: write each language as its name and code, such as English (en); these aren't: %s. Learn more: https://xlsform.org/en/#multiple-language-support", strings.Join(bad, ", "))
}
