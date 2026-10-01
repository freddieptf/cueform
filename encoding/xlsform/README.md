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
missing a required column. `ErrInvalidLabel` means a translatable column
has no `::lang` suffix. `GetLangFromCol` wraps `ErrInvalidLabel`, so
check for it with `errors.Is`.

```go
var TranslatableCols = []string{"label", "required_message", "constraint_message", "hint"}
```

TranslatableCols lists the columns that hold one value per language.
In CUE they are structs that map a language to text. In XLSForm they
are spread across `column::language` headers.

## func IsTranslatableColumn

```go
func IsTranslatableColumn(column string) bool
```

IsTranslatableColumn reports whether column starts with one of the
`TranslatableCols`. It matches by prefix, so `label`, `label::English (en)`
and `hint_extra` all return true.

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
- Translatable columns (`label`, `hint`, `required_message`,
  `constraint_message`) map a language to text. The language key
  becomes the `::` suffix of the XLSForm header.

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
  `field::lang` column per language. Every other field must be a string,
  bool or number:

  | CUE value          | cell     |
  |--------------------|----------|
  | `"text"`           | `text`   |
  | `true` / `false`   | `yes` / `no` |
  | `3`                | `3`      |
  | `1.5`              | `1.5`    |

  A list or struct fails with `<path>: cannot write a list as an xlsform
  cell`. The schema accepts bools for `required` and `read_only`.
- If `type` starts with `select_`, the encoder writes the type column as
  `"<type> <choices.list_name>"`, for example `select_one ages`.
- If `type` starts with `begin_` or `begin `, the encoder writes the
  group row, then each child, then a closing row. The closing row keeps
  the same separator: `begin_repeat` is closed with `end_repeat`, and
  `begin group` with `end group`. If any child fails, the whole encode
  fails.

**Choices sheet.** Written only if at least one choice row exists.

- Each `select_*` question adds a `list_name`, `name` and `label::lang`
  row for every choice, in order.
- The `filterCategory` key in a choice entry is skipped and not
  written.
- Choice lists are not de-duplicated. If two questions use the same
  list, its rows are written twice.

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
the first label column).

## Decoding rules

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
`{<name>: {<lang>: <label>}}` in that list's `choices`. The list is
wrapped as `pkg.#Choices & {...}`. A plain `label` column with no
language returns `ErrInvalidLabel`. Columns other than `name` and
`label::*`, such as filter columns, are dropped.

**Survey.**

- Empty rows are skipped. So are empty cells, which means a column with
  no value in a row produces no field for that row.
- A row whose type starts with `begin` becomes `pkg.#Group & {...}`. The
  rows after it, up to the next row whose type starts with `end`, go
  into its `children`. Groups can be nested.
- Every other row becomes `pkg.#Question & {...}`.
- A type of `select_<x> <list>` is split into `type: "select_<x>"` and a
  `choices` field that holds the decoded `<list>` from the choices
  sheet.
- A translatable header must have the form `col::lang`. It becomes a
  struct `col: {lang: text}`. If it has no language, decoding fails with
  `ErrInvalidLabel`.
- Every other cell becomes a string field named after its header.
- Each top-level element becomes a top-level field named after its
  `name` value. Elements with one field or fewer, such as stray `end`
  rows, are skipped.

**Settings.** If the settings sheet has exactly one data row, it becomes
`form_settings: pkg.#Settings & {type: "settings", ...}`. Every value is
a string. The decoder ignores the settings sheet if it has zero data
rows or more than one.

## Caveats

These describe current behavior. Most are bugs or gaps.

- **The `end` metadata type breaks decoding.** The decoder treats any
  row whose type starts with `end` as the close of a group. An `end`
  metadata question, which the schema allows, ends the enclosing group
  early, or the whole survey if it is at the top level.
- **Types are not round-tripped.** The encoder writes bools as
  `yes`/`no` and numbers as text, and the decoder always outputs
  strings. `read_only: true` therefore decodes as `read_only: "yes"`.
- **`filterCategory` is lost.** The encoder skips it, and the decoder
  never produces it, so it does not survive either direction.
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
- **Prefix matching.** `IsTranslatableColumn` and the decoder's
  required-column check both match by prefix. A column such as
  `hint_extra` is treated as translatable, and `name_foo` satisfies the
  required `name` column.
