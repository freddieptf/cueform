# package xlsform

    import "github.com/freddieptf/cueform/encoding/xlsform"

Package xlsform converts between CUE form definitions and
[XLSForm](https://xlsform.org/en/ref-table/) `.xlsx` workbooks.

An `Encoder` turns a CUE file into an XLSForm workbook. A `Decoder`
turns an XLSForm workbook back into CUE source. The CUE side is a tree
of questions, groups, choice lists and settings. The XLSForm side is
the flat `survey`, `choices` and `settings` sheets.

This document describes the package as it currently behaves, including
known limitations. See [Caveats](#caveats).

## Index

- [Variables](#variables)
- [func IsTranslatableColumn](#func-istranslatablecolumn)
- [func GetLangFromCol](#func-getlangfromcol)
- [func LoadInstance](#func-loadinstance)
- [func LoadValue](#func-loadvalue)
- [type CueForm](#type-cueform)
  - [func ParseCueForm](#func-parsecueform)
- [type Encoder](#type-encoder)
  - [func NewEncoder](#func-newencoder)
  - [func (*Encoder) Encode](#func-encoder-encode)
- [type Decoder](#type-decoder)
  - [func NewDecoder](#func-newdecoder)
  - [func (*Decoder) UsePkg](#func-decoder-usepkg)
  - [func (*Decoder) Decode](#func-decoder-decode)
- [Form model](#form-model)
- [Encoding rules](#encoding-rules)
- [Decoding rules](#decoding-rules)
- [Caveats](#caveats)

## Variables

```go
var (
	ErrInvalidXLSForm      = errors.New("xlsform structure is incorrect")
	ErrInvalidXLSFormSheet = errors.New("found xlsform sheet missing a required column")
	ErrInvalidLabel        = errors.New("found translatable column with no language code")
)
```

The decoder returns these errors. `ErrInvalidXLSForm` means a required
sheet is missing or empty. `ErrInvalidXLSFormSheet` means a sheet is
missing a required column. `ErrInvalidLabel` means a translatable
column header is malformed, such as `label:en` with a single colon.
`GetLangFromCol` wraps `ErrInvalidLabel`, so check for it with
`errors.Is`.

```go
var TranslatableCols = []string{"label", "required_message", "constraint_message", "hint", "guidance_hint", "image", "big-image", "audio", "video"}
```

TranslatableCols lists the columns that can hold one value per language.
In CUE a translated value is a struct that maps a language to text, and
in XLSForm it is spread across `column::language` headers. Any of these
columns can instead hold one plain value, as in a single-language form:
`label: "Name"` is a `label` column with no language. The schema calls
this type `#Text`, `string | #Translatable`.

## func IsTranslatableColumn

```go
func IsTranslatableColumn(column string) bool
```

IsTranslatableColumn reports whether column is one of the
`TranslatableCols`, with or without a `::lang` suffix. It compares the
text before the first `:`, so `label`, `label::English (en)` and
`big-image::fr` return true, and `hint_extra` returns false. A typo such
as `label:en` also returns true, so the decoder rejects it for its
missing language.

## func GetLangFromCol

```go
func GetLangFromCol(translatableColumn string) (col string, lang string, err error)
```

GetLangFromCol splits a header of the form `column::language` into its
two parts. For example, `"label::English (en)"` returns `"label"` and
`"English (en)"`. It returns an error wrapping `ErrInvalidLabel` if the
header has no `::` separator or if the column part is not in
`TranslatableCols`.

## func LoadInstance

```go
func LoadInstance(path string) ([]*build.Instance, error)
```

LoadInstance loads the CUE file at path as a build instance. If the same
directory contains a file named `labels.cue`, that file is loaded into
the same instance. This is how labels extracted by `cueform yank labels`
are resolved at encode time. The load runs with `Dir` set to the form's
directory, so CUE finds `cue.mod` by walking up from the form, wherever
the command runs. For example, `cueform encode sample/scripting/survey.cue`
works from the repository root. Only the form and `labels.cue` are
loaded, not the whole package. That matters when several forms share a
package, as in `sample/composition`.

## func LoadValue

```go
func LoadValue(path string) (*cue.Value, error)
```

LoadValue loads path with `LoadInstance`, builds it and checks that the
result is fully concrete. It returns an error if loading, building or
validating fails, with the CUE error details included in the message.

## type CueForm

```go
type CueForm struct {
	SurveyElements []*cue.Value
	Settings       *cue.Value
}
```

CueForm is a loaded CUE form split into its parts. SurveyElements holds
the top-level questions and groups in source order. Settings holds the
`form_settings` field, or nil if there is none.

### func ParseCueForm

```go
func ParseCueForm(file string) (*CueForm, error)
```

ParseCueForm loads file with `LoadValue` and walks its top-level fields
in order. Regular fields only: the walk skips hidden fields (`_x`) and
definitions (`#X`). The field named `form_settings` becomes
`Settings`. Every other field is added to `SurveyElements`.

## type Encoder

```go
type Encoder struct{}
```

Encoder converts CUE forms to XLSForm workbooks. It has no
configuration.

### func NewEncoder

```go
func NewEncoder() *Encoder
```

### func (*Encoder) Encode

```go
func (encoder *Encoder) Encode(filePath string) (*bytes.Buffer, error)
```

Encode loads the CUE form at filePath with `ParseCueForm`, validates
each survey element against the embedded schema and returns the `.xlsx`
workbook bytes. See [Encoding rules](#encoding-rules).

```go
buf, err := xlsform.NewEncoder().Encode("survey.cue")
if err != nil {
	log.Fatal(err)
}
os.WriteFile("survey.xlsx", buf.Bytes(), 0o644)
```

## type Decoder

```go
type Decoder struct {
	// contains unexported fields
}
```

Decoder converts XLSForm workbooks to CUE source. The output imports a
schema package, and the decoder needs the import path of that package.
The package must define `#Question`, `#Group`, `#Choices` and
`#Settings`.

### func NewDecoder

```go
func NewDecoder(pkg string) *Decoder
```

NewDecoder returns a Decoder that imports the schema package at import
path pkg, for example `"github.com/freddieptf/cueform/xlsform"`. The
identifier used in the generated code is the last element of the path,
`xlsform` in that example.

### func (*Decoder) UsePkg

```go
func (d *Decoder) UsePkg(schemaPkg string)
```

UsePkg changes the schema package used by later calls to Decode.

### func (*Decoder) Decode

```go
func (d *Decoder) Decode(r io.Reader) ([]byte, error)
```

Decode reads an `.xlsx` workbook from r and returns formatted CUE source
in `package main`. The source is formatted with `format.Simplify`. See
[Decoding rules](#decoding-rules).

```go
f, _ := os.Open("survey.xlsx")
src, err := xlsform.NewDecoder("github.com/freddieptf/cueform/xlsform").Decode(f)
if err != nil {
	log.Fatal(err)
}
os.WriteFile("survey.cue", src, 0o644)
```

## Form model

Both directions use the same CUE shape:

```cue
family_name: xlsform.#Question & {
	type: "text"
	name: "family_name"
	label: "English (en)": "What's your family name?"
}

father: xlsform.#Group & {
	type: "begin_group"
	name: "father"
	label: "English (en)": "Father"
	children: [
		xlsform.#Question & {
			type: "select_one"
			name: "age"
			label: "English (en)": "How old is your father?"
			choices: xlsform.#Choices & {
				list_name: "ages"
				choices: [
					{over_30: "English (en)": "Over 30"},
					{over_40: "English (en)": "Over 40"},
				]
			}
		},
	]
}

form_settings: xlsform.#Settings & {
	type:             "settings"
	form_title:       "test"
	form_id:          "test_id"
	version:          "1"
	default_language: "English (en)"
}
```

- Each top-level field (except `form_settings`) is one survey element,
  in source order. The encoder ignores the field's own label; it uses
  the `name` inside the element.
- A group lists its nested elements in `children`.
- A `select_*` question holds its choice list in `choices`. Each entry
  in `choices.choices` is a struct whose key is the choice `name` and
  whose value maps a language to the choice label.
- Translatable columns (`label`, `hint`, `guidance_hint`,
  `required_message`, `constraint_message`, and the media columns
  `image`, `big-image`, `audio` and `video`) map a language to text. The
  language key becomes the `::` suffix of the XLSForm header. In a
  single-language form any of them, and choice labels, can be a plain
  value written to a column with no language: `label: "Name"`,
  `{yes: "Yes"}`.

## Encoding rules

**Validation.** Before writing anything, the encoder checks each survey
element, including every nested child, against the schema embedded from
`schema/xlsform/schema.cue` (package
`github.com/freddieptf/cueform/schema`). This happens whatever
definitions the form itself imports. Types that start with `begin_` or
`begin ` are checked against `#Group`; everything else is checked
against `#Question`.

- `type` is checked first, against `#QuestionType` or `#GroupType`. An
  unknown type fails with `<path>: "txt" is not a valid question type`.
- The whole element is then unified with its definition and must be
  concrete. Failures name the element's path, such as
  `father.children[0] does not match the schema`, followed by the CUE
  error details.
- `choices` is required on `select_one`, `select_multiple` and `rank`
  questions; without it, the element fails with
  `#Question.choices: field is required but not present`.
- `label` is required except on types that are never shown:
  `calculate`, `hidden`, `background-audio`, `xml-external`,
  `csv-external` and the metadata types (`start`, `end`, `today`,
  `deviceid`, …).
- Both definitions end in `...`, because XLSForm allows extra columns.
  Unknown fields are accepted, so a misspelled optional field is not
  caught.
- `form_settings` is not validated.

**Survey sheet.** Elements are written depth-first in source order.

- Every element must have a string `type`.
- Every field except `children` and `choices` becomes a column.
  Translatable fields are decoded as `{lang: text}` and become one
  `field::lang` column per language. Every other field must be a CUE
  string, bool or number. All are written as text cells, which is how
  pyxform reads every cell anyway:

  | CUE value | cell |
  |-----------|------|
  | `"text"` | `text`, exactly as it is |
  | `true` / `false` | `yes` / `no` |
  | `3`, `len(_sizes)` | `3` |
  | `1.50` | `1.50`, CUE's exact decimal with its trailing zeros |

  The schema still types some fields. `required` and `read_only` take a
  string or bool, so `required: true` works. Fields such as `relevant`
  are strings only, and anything else fails with
  `<path> does not match the schema`. Untyped fields such as
  `repeat_count`, `default` and the settings `version` take numbers. A
  list or struct fails with
  `<path>: xlsform values must be strings, bools or numbers`.
- If `type` starts with `select_` or is `rank`, the encoder writes the
  type column as `"<type> <choices.list_name>"`, for example
  `select_one ages` or `rank ages`.
- `or_other` isn't supported. pyxform adds its "other" choice to the
  shared list, so every question using the list shows it. Add an
  "other" choice and a text question with `relevant`, as the XLSForm spec
  recommends. An `or_other` field fails with `<path>: or_other is not
  supported` (`ErrOrOther`).
- If `type` starts with `begin_` or `begin `, the encoder writes the
  group row, then each child, then a closing row. The closing row keeps
  the same separator: `begin_repeat` is closed with `end_repeat`, and
  `begin group` with `end group`. If any child fails, the whole encode
  fails.

**Choices sheet.** Written only if at least one choice row exists.

- Each `select_*` or `rank` question adds a `list_name`, `name` and
  `label::lang` row for every choice, in order.
- Each list is written once, however many questions use it. pyxform
  rejects a list whose choice names repeat. If two questions use the same
  `list_name` with different choices, encoding fails with
  `<path>: choice list "yes_no" differs from the one at <first path>; give one of them another list_name`.
- An entry's `filterCategory` becomes extra columns on each of its
  choices, which a question's `choice_filter` can test. For example,
  `{nairobi: en: "Nairobi", filterCategory: country: "ke"}` writes `ke`
  in a `country` column, and `choice_filter: "country=${country}"`
  makes a cascading select. A filter can't use a column XLSForm defines
  for choices (`list_name`, `name`, `label`, `image`, `big-image`,
  `audio`, `video`, `media`); that fails with
  `<path>: "image" is a choices sheet column, not a filter`.

**Settings sheet.** Written only if `form_settings` exists. Its fields
become a single row. The `type` field is removed.

**Column order.** In each sheet, known columns come first in this
order:

| sheet    | known columns, in order |
|----------|-------------------------|
| survey   | `type`, `name`, `label`, `required`, `required_message`, `relevant`, `repeat_count`, `constraint`, `constraint_message`, `hint`, `choice_filter`, `read_only`, `calculation`, `appearance`, `default` |
| choices  | `list_name`, `name`, `label` |
| settings | `form_title`, `form_id`, `public_key`, `submission_url`, `default_language`, `style`, `version`, `instance_name` |

A translatable column's `col::lang` headers sit where `col` would be,
sorted by language. Any other columns are added at the end in
alphabetical order. Only columns that some element uses are written.

**Workbook.** The default `Sheet1` is deleted. Column width is set to 30
for every column, and to 50 for column C of the survey sheet (usually
the first label column). Each sheet's dimension is set to the range
written. pyxform reads workbooks with openpyxl in read-only mode, which
reads only the cells inside the dimension. With excelize's default of
`A1`, pyxform produces an XForm with no questions.

## Decoding rules

**Cell values.** XLSForm values are text, and cells are read as pyxform
4.5.0 reads them. This keeps imports faithful: a decoded spreadsheet
encodes back to one that pyxform reads the same way. The same rules
apply in every sheet.

- Text and bool cells read as they are: `3.00` as text stays `"3.00"`,
  and bools read `TRUE`/`FALSE`.
- Number cells (`repeat_count`, `default`, numeric versions and choice
  names) use their stored value, not the text Excel displays
  (excelize's default):

  | number cell | Excel shows | decoded as |
  |---|---|---|
  | `3` formatted `0.00` | `3.00` | `"3"` |
  | `0.5` formatted `0.00%` | `50.00%` | `"0.5"` |
  | `1234.5` formatted `#,##0.000` | `1,234.500` | `"1234.5"` |
  | `0.000123` formatted `0.00` | `0.00` | `"0.000123"` |

- Date and time cells decode as pyxform writes them: `"2024-01-15 00:00:00"`,
  or `"13:30:00"` for a time on its own. Each one also logs a warning
  naming the cell:
  `warning: survey!D2 (hint::en): Excel stored this as a date, which pyxform reads as "2026-01-02 00:00:00"; format the cell as text if that isn't what you meant`.
  XLSForm values are text, so a date cell is usually text that Excel
  converted. A hint typed as `1/2` becomes 2 January, and pyxform puts
  `2026-01-02 00:00:00` in the form without warning. Matching pyxform
  keeps imports working and faithful, and the warning points at the cell
  to fix. A cell counts as a date if its number format's first section
  has a `d`, `m`, `h`, `y` or `s` outside quoted text and `[...]`
  sections, as in openpyxl.
- Decoded values are always CUE strings, apart from the yes/no bools
  below.

`testdata/pyxform/cells.xlsx` holds these cases, written by openpyxl,
and `cells.json` holds pyxform's reading of them. `TestCellsMatchPyxform`
checks the decoder against that reading. Set `CUEFORM_PYXFORM_PYTHON` to
a Python with `pyxform==4.5.0` installed, and
`TestRoundTripMatchesPyxform` will also run pyxform itself. It checks
that `cells.json` is current, and that decoding and encoding again gives
a workbook pyxform reads the same way.

**Sheet validation.**

- The `survey` sheet is required. Its header row must contain columns
  that start with `type`, `name` and `label`, so `label::en` counts as
  `label`.
- The `choices` sheet is optional. If it exists, it must contain
  `list_name`, `name` and `label` columns, matched the same way.
- The `settings` sheet is optional. If it exists, it must not be empty.
- A required sheet that is missing or empty returns `ErrInvalidXLSForm`.
  A missing required column returns `ErrInvalidXLSFormSheet`. If an
  optional sheet is missing, the decoder logs that and keeps going.

**Choices.** Rows are grouped by `list_name`. Each row becomes
`{<name>: {<lang>: <label>}}` in that list's `choices`, or
`{<name>: <label>}` when the sheet has a single plain `label` column.
The list is wrapped as `pkg.#Choices & {...}`. Other columns with a
value become
the entry's `filterCategory`, as in
`{nairobi: en: "Nairobi", filterCategory: country: "ke"}`. Media columns
(`image`, `audio`, `video`, `big-image`, `media::*`) are dropped.

**Survey.**

- Empty rows are skipped. So are empty cells, which means a column with
  no value in a row produces no field for that row.
- A row whose type starts with `begin` becomes `pkg.#Group & {...}`. The
  rows after it, up to the next row whose type starts with `end`, go
  into its `children`. Groups can be nested.
- Every other row becomes `pkg.#Question & {...}`.
- A type of `select_<x> <list>` or `rank <list>` is split into
  `type: "select_<x>"` (or `"rank"`) and a `choices` field that holds the
  decoded `<list>` from the choices sheet. A suffix after the list name,
  such as `or_other`, fails with `ErrOrOther`.
- A translatable header `col::lang` becomes a struct
  `col: {lang: text}`. A plain `col` header becomes a plain value,
  `label: "Name"`, unless the sheet also has `col::lang` headers. In that
  case it is pyxform's `default` language: `label` with `label::fr`
  decodes as `label: {default: "Name", fr: "Nom"}`. A malformed header
  such as `label:en` fails with `ErrInvalidLabel`.
- In the `required` and `read_only` columns, the exact values `yes`,
  `Yes`, `YES`, `true`, `True` and `TRUE` become `true`, and `no`, `No`,
  `NO`, `false`, `False` and `FALSE` become `false`. This is the same
  list pyxform uses (`aliases.BINDING_CONVERSIONS`, v4.5.0). Any other
  value, including `true()` and expressions such as `${age} >= 18`,
  stays a string.
- Every other cell becomes a string field named after its header, with
  its value read as described in [Cell values](#decoding-rules).
- Each top-level element becomes a top-level field named after its
  `name` value. Elements with one field or fewer, such as stray `end`
  rows, are skipped.

**Settings.** If the settings sheet has exactly one data row, it becomes
`form_settings: pkg.#Settings & {type: "settings", ...}`. Every value is
a string. The decoder ignores the settings sheet if it has zero data
rows or more than one.

## Caveats

These describe current behavior. Most are bugs or gaps.

- **Yes/no spellings are normalized.** `TRUE`, `YES`, `false` and the
  other spellings in `required` and `read_only` decode as CUE bools, and
  bools encode as `yes`/`no`. A sheet that used `TRUE` therefore comes
  back with `yes`. pyxform reads both as `true()`, so the form is the
  same.

- **The `end` metadata type breaks decoding.** The decoder treats any
  row whose type starts with `end` as the close of a group. An `end`
  metadata question, which the schema allows, ends the enclosing group
  early, or the whole survey if it is at the top level.
- **Numbers come back as strings.** `repeat_count: 3` encodes as `3` and
  decodes as `repeat_count: "3"`. pyxform reads both the same way.
- **Unusual date cells differ from pyxform.** No XLSForm column needs
  these, so they aren't matched:
  - durations (`[h]:mm:ss`), which pyxform writes as `1 day, 6:00:00`;
  - workbooks using the 1904 date system;
  - dates before March 1900;
  - times with fractions of a second.
- **Unusual numbers differ from pyxform.** Numbers below 0.0001 decode
  as `0.00001`, where pyxform writes `1e-05`, and integers beyond 2^53
  lose digits.
- **Choice media is lost.** The encoder can't write choice `image`,
  `audio` or `video` columns, and the decoder drops them.
- **Choice list with no match.** The decoder doesn't check that a
  `select_*` list name exists in the choices sheet. This includes
  `select_one_from_file <file>`. With no match, `choices` gets a nil
  expression.
- **Possible panics.** The decoder dereferences values without
  checking them first, so these inputs can panic:
  - a top-level element with no `name` column value;
  - a settings row with fewer cells than the header row;
  - a row whose `type` cell is beyond the end of that row.
- **An early `end` stops decoding.** An `end` row at the top level,
  with no matching `begin`, ends the survey. Rows after it are dropped.
- **Choice-list order is not fixed.** Choice lists are built from a
  Go map. That doesn't change the output, because each list is
  attached to the question that uses it, but it matters if you add code
  that emits the lists on their own.
- **Prefix matching.** The decoder's required-column check matches by
  prefix, so `name_foo` satisfies the required `name` column.
