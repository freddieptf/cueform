package xlsform

import (
	"bufio"
	_ "embed"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// The XLSForm spec recommends naming each language with a two-letter code after it, as in
// label::English (en), so apps can match the form's language to the device's. The encoder warns
// about every language not written that way.

//go:embed iana_subtags/iana_subtags_2_characters.txt
var twoLetterSubtagsFile string

var (
	// a name, one space, and a two-letter code in brackets, as in "English (en)"
	recommendedLanguageRe = regexp.MustCompile(`^\S(.*\S)? \(([a-z]{2})\)$`)

	twoLetterSubtags = sync.OnceValue(func() map[string]bool {
		tags := map[string]bool{}
		scanner := bufio.NewScanner(strings.NewReader(twoLetterSubtagsFile))
		for scanner.Scan() {
			tags[strings.TrimSpace(scanner.Text())] = true
		}
		return tags
	})
)

// languagesNotRecommended returns the languages not written as Name (code) with a two-letter
// code from the IANA registry. "default" is skipped: it is pyxform's name for a column with no
// language next to translated ones, not a name the author chose.
func languagesNotRecommended(languages []string) []string {
	var bad []string
	for _, lang := range languages {
		if lang == "default" {
			continue
		}
		match := recommendedLanguageRe.FindStringSubmatch(lang)
		if match == nil || !twoLetterSubtags()[match[2]] {
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
	return fmt.Sprintf("warning: write each language as its name and two-letter code, such as English (en); these aren't: %s. Learn more: https://xlsform.org/en/#multiple-language-support", strings.Join(bad, ", "))
}
