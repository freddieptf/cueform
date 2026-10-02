package xlsform

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"cuelang.org/go/cue"
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

// checkDefaultLanguage requires form_settings.default_language when the form has more than one
// language, and requires it to be one of them. pyxform marks no translation as the default in
// either case, so the app picks one. A plain column next to translated ones is pyxform's
// "default" language, which is the default already.
func checkDefaultLanguage(settings *cue.Value, languages []string) error {
	var defaultLanguage string
	if settings != nil {
		if v := settings.LookupPath(cue.ParsePath("default_language")); v.Exists() {
			var err error
			if defaultLanguage, err = v.String(); err != nil {
				return fmt.Errorf("%s: %w", v.Path(), err)
			}
		}
	}
	named := slices.DeleteFunc(slices.Clone(languages), func(l string) bool { return l == "default" })
	switch {
	case defaultLanguage == "" && len(named) > 1 && !slices.Contains(languages, "default"):
		return fmt.Errorf("form_settings.default_language is required, because the form has more than one language: %s", strings.Join(named, ", "))
	case defaultLanguage != "" && len(named) > 0 && !slices.Contains(languages, defaultLanguage):
		return fmt.Errorf("form_settings.default_language %q isn't one of the form's languages: %s", defaultLanguage, strings.Join(named, ", "))
	}
	return nil
}
