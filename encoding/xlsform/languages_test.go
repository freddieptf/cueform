package xlsform

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
)

// languages must be written as the spec recommends: a name, one space, and a code in brackets;
// any code is accepted
func TestLanguagesNotRecommended(t *testing.T) {
	for lang, ok := range map[string]bool{
		"English (en)":            true,
		"Español (es)":            true,
		"Swahili (sw)":            true,
		"Chinese Simplified (zh)": true,
		// pyxform's name for a plain column next to translated ones
		"default":           true,
		"English":           false,
		"en":                false,
		"Dutch(nl)":         false,
		"Dutch  (nl)":       false,
		" Dutch (nl)":       false,
		"Dutch (NL)":        true,
		"Luo (luo)":         true,
		"Chinese (zh-Hans)": true,
		"Foo (xx)":          true,
		"Dutch ()":          false,
		"Dutch (n l)":       false,
		"(en)":              false,
	} {
		bad := languagesNotRecommended([]string{lang})
		if (len(bad) == 0) != ok {
			t.Errorf("%q: have %v, want recommended=%v", lang, bad, ok)
		}
	}
}

func TestEncodeLanguageWarning(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	// form_media.cue uses the languages en and fr
	if _, err := NewEncoder().Encode("testdata/form_media.cue"); err != nil {
		t.Fatal(err)
	}
	if want := "these aren't: en, fr."; !strings.Contains(logs.String(), want) {
		t.Errorf("want %q in %q", want, logs.String())
	}

	logs.Reset()
	// form.cue uses English (en)
	if _, err := NewEncoder().Encode("testdata/form.cue"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logs.String(), "name and code") {
		t.Errorf("unexpected warning %q", logs.String())
	}
}

// a form with more than one language needs form_settings.default_language, set to one of them
func TestEncodeDefaultLanguage(t *testing.T) {
	for file, want := range map[string]string{
		"testdata/form_languages_no_default.cue":    "form_settings.default_language is required, because the form has more than one language: English (en), Swahili (sw)",
		"testdata/form_languages_bad_default.cue":   `form_settings.default_language "French (fr)" isn't one of the form's languages: English (en), Swahili (sw)`,
		"testdata/form_languages_plain_default.cue": "",
		"testdata/form_languages_one.cue":           "",
		"testdata/form_media.cue":                   "",
	} {
		_, err := NewEncoder().Encode(file)
		switch {
		case want == "" && err != nil:
			t.Errorf("%s: %v", file, err)
		case want != "" && (err == nil || !strings.Contains(err.Error(), want)):
			t.Errorf("%s: have %v, want %q", file, err, want)
		}
	}
}
